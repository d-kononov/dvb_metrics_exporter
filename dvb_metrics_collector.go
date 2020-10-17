package main

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/ziutek/dvb/linuxdvb/frontend"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	adapterPrefix  = "adapter"
	frontendPrefix = "frontend"
)

type dvbCollector struct {
	adapters     []os.FileInfo
	adapterInfos []*frontendInfo
}

type frontendInfo struct {
	ber         int
	device      *frontend.Device
	info        string
	lock        bool
	mode        string
	adapterNum  string
	frontendNum string
	path        string
	signal      int
	snr         int
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

	defer func() {
		for _, adapterInfo := range c.adapterInfos {
			_ = adapterInfo.device.Close()
		}
	}()

	for _, f := range c.adapters {
		adapterPath := filepath.Join(devDvbPath, f.Name())
		adapterNum := f.Name()[len(adapterPrefix):]

		frontends, err := ioutil.ReadDir(adapterPath)
		if err != nil {
			log.WithError(err).Errorf("failed to list frontends of adapter %s", adapterPath)
			continue
		}

		for _, frontendFile := range frontends {
			if !strings.HasPrefix(frontendFile.Name(), frontendPrefix) {
				continue
			}

			frontendPath := filepath.Join(adapterPath, frontendFile.Name())
			frontendNum := frontendFile.Name()[len(frontendPrefix):]
			log.Debugf("adding frontend %s", frontendPath)
			if err != nil {
				log.WithError(err).Errorf("failed to get adapter number of frontend %s", frontendPath)
				continue
			}
			fe, err := frontend.OpenRO(frontendPath)
			if err != nil {
				log.WithError(err).Errorf("failed to open %s frontend", frontendPath)
				continue
			}

			c.adapterInfos = append(
				c.adapterInfos,
				&frontendInfo{
					device:      &fe,
					adapterNum:  adapterNum,
					frontendNum: frontendNum,
					path:        frontendPath,
				},
			)
		}
	}

	for {
		wg := sync.WaitGroup{}
		wg.Add(len(c.adapterInfos))

		for i := 0; i < len(c.adapterInfos); i++ {
			adapterInfo := c.adapterInfos[i]
			go func() {
				defer wg.Done()
				logger := log.WithFields(log.Fields{"adapter-num": adapterInfo.adapterNum, "adapter-path": adapterInfo.path})

				if *apiV5Force {
					deliverySystem := "unknown"
					mode, err := adapterInfo.device.DeliverySystem()
					if err != nil {
						log.WithError(err).Debugf("failed to parse delivery system")
					} else {
						deliverySystem = mode.String()
					}
					stat, errStat := adapterInfo.device.Stat()
					if errStat == nil {
						if len(stat.CNR) > 0 && len(stat.Signal) > 0 {
							adapterInfo.snr = int(stat.CNR[0].Relative()) * *snrCorrection
							adapterInfo.signal = int(stat.Signal[0].Relative())
							adapterInfo.mode = deliverySystem

							if len(stat.PreErrBit) > 0 {
								adapterInfo.ber = int(stat.PreErrBit[0].Counter())
							}

							if adapterInfo.snr == 0 {
								adapterInfo.lock = false
							} else {
								adapterInfo.lock = true
							}
						}
					}
				} else {
					apiV3 := frontend.API3{Device: *adapterInfo.device}
					status, err := apiV3.Status()
					if err != nil {
						logger.WithError(err).Error("failed to get status")
						adapterInfo.lock = false
					} else {
						adapterInfo.lock = strings.Contains(status.String(), "+lock")
					}

					adapterInfo.device.Stat()
					if !adapterInfo.lock {
						logger.Debug("no lock!")
						return
					}

					deliverySystem, err := apiV3.DeliverySystem()
					if err != nil {
						logger.WithError(err).Error("failed to get DeliverySystem")
					} else {
						adapterInfo.mode = deliverySystem.String()
					}

					signalStrength, err := apiV3.SignalStrength()
					if err != nil {
						logger.WithError(err).Error("failed to get SignalStrength")
					} else {
						ss, _ := strconv.ParseUint(fmt.Sprintf("%x", uint16(signalStrength)), 16, 64)
						if adapterInfo.mode == "DVB-T2" {
							adapterInfo.signal = int(ss)
						} else {
							adapterInfo.signal = int((ss * 100) / signalSnrMaxNumber)
							if adapterInfo.signal > 97 {
								if signalStrength < 1 {
									adapterInfo.signal = int(signalStrength*-1) * *snrCorrection
								} else {
									adapterInfo.signal = int(signalStrength) * *snrCorrection
								}
							}
						}
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

					// USE API V5 as a backup plan
					mode, _ := adapterInfo.device.DeliverySystem()
					if adapterInfo.snr == 0 { //|| adapterInfo.mode == "DVB-T2" {
						stat, err := adapterInfo.device.Stat()
						if err == nil {
							if len(stat.CNR) > 0 {
								adapterInfo.snr = int(stat.CNR[0].Relative()) * *snrCorrection
								adapterInfo.signal = int(stat.Signal[0].Relative())
								adapterInfo.mode = mode.String()
								log.Info(stat.ErrBlk)
								log.Info(stat.PostErrBit)
								log.Info(stat.PreErrBit)
								log.Info(stat.TotBlk)
							}
						}
					}
				}

				logger.Debugf("collected, lock: %t signal: %d, snr: %d, ber: %d, mode: %s",
					adapterInfo.lock, adapterInfo.signal, adapterInfo.snr, adapterInfo.ber, adapterInfo.mode)
			}()
		}
		wg.Wait()
		time.Sleep(interval)
	}
}
