package mount

import (
	"syscall"
	"time"
	"github.com/Sirupsen/logrus"
)

func mount(device, target, mType string, flag uintptr, data string) error {
	ts := time.Now()
	if err := syscall.Mount(device, target, mType, flag, data); err != nil {
		return err
	}
	logrus.Infof("LATENCY in (pkg/mount/mounter_linux.go) Mount for %v in %v", target, time.Since(ts))

	// If we have a bind mount or remount, remount...
	if flag&syscall.MS_BIND == syscall.MS_BIND && flag&syscall.MS_RDONLY == syscall.MS_RDONLY {
		err := syscall.Mount(device, target, mType, flag|syscall.MS_REMOUNT, data)
		logrus.Infof("LATENCY in (pkg/mount/mounter_linux.go) re-Mount for %v in %v", target, time.Since(ts))
		return err
	}
	return nil
}

func unmount(target string, flag int) error {
	return syscall.Unmount(target, flag)
}
