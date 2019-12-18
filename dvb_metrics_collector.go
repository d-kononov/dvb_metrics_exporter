package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/ziutek/dvb/linuxdvb/frontend"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type dvbCollector struct {
	adapters     []os.FileInfo
	adapterInfos map[int]*adapterInfo
}

type adapterInfo struct {
	ber    int
	device *frontend.Device
	info   string
	lock   bool
	mode   string
	num    int
	path   string
	signal int
	snr    int
}

func (c *dvbCollector) startCollectMetrics() {
	interval, err := time.ParseDuration(*collectInterval)
	if err != nil {
		log.WithError(err).Errorf("failed to parse 'collectInterval=%v', using default interval", *collectInterval)
		interval = 5 * time.Second
	}
	snrMaxNumber := signalSnrMaxNumber
	if *snrCorrection > 1 && *snrCorrection < signalSnrMaxNumber {
		snrMaxNumber = signalSnrMaxNumber / *snrCorrection
	}

	if c.adapterInfos == nil {
		c.adapterInfos = map[int]*adapterInfo{}
	}

	defer func() {
		for _, adapterInfo := range c.adapterInfos {
			_ = adapterInfo.device.Close()
		}
	}()

	for _, f := range c.adapters {
		adapterPath := filepath.Join(devDvbPath, f.Name(), "frontend0")
		log.Debugf("adding adapter %s", adapterPath)
		adapterNum, err := strconv.Atoi(f.Name()[len(f.Name())-1:])
		if err != nil {
			log.WithError(err).Errorf("failed to get adapter number of adapter %s", adapterPath)
			continue
		}
		if _, ok := c.adapterInfos[adapterNum]; !ok {
			fe, err := frontend.OpenRO(adapterPath)
			if err != nil {
				log.WithError(err).Errorf("failed to open %s adapter", adapterPath)
				continue
			}
			c.adapterInfos[adapterNum] = &adapterInfo{device: &fe, num: adapterNum, path: adapterPath}
		}
	}

	for {
		wg := sync.WaitGroup{}
		wg.Add(len(c.adapterInfos))

		for i := 0; i < len(c.adapterInfos); i++ {
			adapterInfo := c.adapterInfos[i]
			go func() {
				defer wg.Done()
				logger := log.WithFields(log.Fields{"adapter-num": adapterInfo.num, "adapter-path": adapterInfo.path})

				apiV3 := frontend.API3{Device: *adapterInfo.device}
				status, err := apiV3.Status()
				if err != nil {
					logger.WithError(err).Error("failed to get status")
					adapterInfo.lock = false
				} else {
					adapterInfo.lock = strings.Contains(status.String(), "+lock")
				}

				if !adapterInfo.lock {
					logger.Debug("no lock!")
					return
				}
				signalStrength, err := apiV3.SignalStrength()
				if err != nil {
					logger.WithError(err).Error("failed to get SignalStrength")
				} else {
					ss, _ := strconv.ParseUint(fmt.Sprintf("%x", uint16(signalStrength)), 16, 64)
					adapterInfo.signal = int((ss * 100) / signalSnrMaxNumber)
				}

				snr, err := apiV3.SNR()
				if err != nil {
					logger.WithError(err).Error("failed to get SNR")
				} else {
					adapterInfo.snr = (int(snr) * 100) / snrMaxNumber
				}

				ber, err := apiV3.BER()
				if err != nil {
					logger.WithError(err).Error("failed to get BER")
				} else {
					adapterInfo.ber = int(ber)
				}
				deliverySystem, err := apiV3.DeliverySystem()
				if err != nil {
					logger.WithError(err).Error("failed to get DeliverySystem")
				} else {
					adapterInfo.mode = deliverySystem.String()
				}

				logger.Debugf("collected, signal: %d, snr: %d, ber: %d, mode: %s", adapterInfo.signal, adapterInfo.snr, adapterInfo.ber, adapterInfo.mode)
			}()
		}
		wg.Wait()
		time.Sleep(interval)
	}
}
