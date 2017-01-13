package daemon

import (
	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/container"
	derr "github.com/docker/docker/errors"
	"github.com/docker/docker/image"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/pkg/idtools"
	"github.com/docker/docker/pkg/stringid"
	volumestore "github.com/docker/docker/volume/store"
	"github.com/docker/engine-api/types"
	containertypes "github.com/docker/engine-api/types/container"
	networktypes "github.com/docker/engine-api/types/network"
	"github.com/opencontainers/runc/libcontainer/label"
	"time"
)

// ContainerCreate creates a container.
func (daemon *Daemon) ContainerCreate(params types.ContainerCreateConfig) (types.ContainerCreateResponse, error) {
	if params.Config == nil {
		return types.ContainerCreateResponse{}, derr.ErrorCodeEmptyConfig
	}

	warnings, err := daemon.verifyContainerSettings(params.HostConfig, params.Config)
	if err != nil {
		return types.ContainerCreateResponse{Warnings: warnings}, err
	}

	err = daemon.verifyNetworkingConfig(params.NetworkingConfig)
	if err != nil {
		return types.ContainerCreateResponse{}, err
	}

	if params.HostConfig == nil {
		params.HostConfig = &containertypes.HostConfig{}
	}
	err = daemon.adaptContainerSettings(params.HostConfig, params.AdjustCPUShares)
	if err != nil {
		return types.ContainerCreateResponse{Warnings: warnings}, err
	}

	container, err := daemon.create(params)
	if err != nil {
		return types.ContainerCreateResponse{Warnings: warnings}, daemon.imageNotExistToErrcode(err)
	}

	return types.ContainerCreateResponse{ID: container.ID, Warnings: warnings}, nil
}

// Create creates a new container from the given configuration with a given name.
func (daemon *Daemon) create(params types.ContainerCreateConfig) (retC *container.Container, retErr error) {
	ts := time.Now()
	var (
		container *container.Container
		img       *image.Image
		imgID     image.ID
		err       error
	)

	if params.Config.Image != "" {
		img, err = daemon.GetImage(params.Config.Image)
		if err != nil {
			return nil, err
		}
		imgID = img.ID()
	}
	logrus.Infof("LATENCY in (daemon/create.go#create) for %v in %v", params.Name, time.Since(ts))

	if err := daemon.mergeAndVerifyConfig(params.Config, img); err != nil {
		return nil, err
	}

	ts = time.Now()
	if container, err = daemon.newContainer(params.Name, params.Config, imgID); err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			if err := daemon.ContainerRm(container.ID, &types.ContainerRmConfig{ForceRemove: true}); err != nil {
				logrus.Errorf("Clean up Error! Cannot destroy container %s: %v", container.ID, err)
			}
		}
	}()
	logrus.Infof("LATENCY in (daemon/create.go#create) newContainer for %v in %v", imgID, time.Since(ts))

	ts = time.Now()
	if err := daemon.setSecurityOptions(container, params.HostConfig); err != nil {
		return nil, err
	}
	logrus.Infof("LATENCY in (daemon/create.go#create) setSecurityOptions for %v in %v", imgID, time.Since(ts))

	ts = time.Now()
	// Set RWLayer for container after mount labels have been set
	if err := daemon.setRWLayer(container); err != nil {
		return nil, err
	}
	logrus.Infof("LATENCY in (daemon/create.go#create) setRWLayer for %v in %v", imgID, time.Since(ts))

	ts = time.Now()
	if err := daemon.Register(container); err != nil {
		return nil, err
	}
	logrus.Infof("LATENCY in (daemon/create.go#create) Register for %v in %v", imgID, time.Since(ts))

	rootUID, rootGID, err := idtools.GetRootUIDGID(daemon.uidMaps, daemon.gidMaps)
	if err != nil {
		return nil, err
	}
	if err := idtools.MkdirAs(container.Root, 0700, rootUID, rootGID); err != nil {
		return nil, err
	}

	if err := daemon.setHostConfig(container, params.HostConfig); err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			if err := daemon.removeMountPoints(container, true); err != nil {
				logrus.Error(err)
			}
		}
	}()

	if err := daemon.createContainerPlatformSpecificSettings(container, params.Config, params.HostConfig); err != nil {
		return nil, err
	}

	var endpointsConfigs map[string]*networktypes.EndpointSettings
	if params.NetworkingConfig != nil {
		endpointsConfigs = params.NetworkingConfig.EndpointsConfig
	}

	if err := daemon.updateContainerNetworkSettings(container, endpointsConfigs); err != nil {
		return nil, err
	}

	ts = time.Now()
	if err := container.ToDiskLocking(); err != nil {
		logrus.Errorf("Error saving new container to disk: %v", err)
		return nil, err
	}
	daemon.LogContainerEvent(container, "create")
	logrus.Infof("LATENCY out (daemon/create.go#create) for %v in %v", imgID, time.Since(ts))
	return container, nil
}

func (daemon *Daemon) generateSecurityOpt(ipcMode containertypes.IpcMode, pidMode containertypes.PidMode, privileged bool) ([]string, error) {
	if ipcMode.IsHost() || pidMode.IsHost() || privileged {
		return label.DisableSecOpt(), nil
	}
	if ipcContainer := ipcMode.Container(); ipcContainer != "" {
		c, err := daemon.GetContainer(ipcContainer)
		if err != nil {
			return nil, err
		}

		return label.DupSecOpt(c.ProcessLabel), nil
	}
	return nil, nil
}

func (daemon *Daemon) setRWLayer(container *container.Container) error {
	logrus.Infof("LATENCY out (daemon/create.go#setRWLayer) enter for %v", container.Name)
	ts := time.Now()
	var layerID layer.ChainID
	if container.ImageID != "" {
		img, err := daemon.imageStore.Get(container.ImageID)
		if err != nil {
			return err
		}
		layerID = img.RootFS.ChainID()
	}
	logrus.Infof("LATENCY out (daemon/create.go#setRWLayer) imageStore.Get for %v in %v", container.Name, time.Since(ts))
	ts = time.Now()
	rwLayer, err := daemon.layerStore.CreateRWLayer(container.ID, layerID, container.MountLabel, daemon.setupInitLayer)
	if err != nil {
		return err
	}
	container.RWLayer = rwLayer
	logrus.Infof("LATENCY out (daemon/create.go#setRWLayer) CreateRWLayer for %v in %v", container.Name, time.Since(ts))

	return nil
}

// VolumeCreate creates a volume with the specified name, driver, and opts
// This is called directly from the remote API
func (daemon *Daemon) VolumeCreate(name, driverName string, opts map[string]string) (*types.Volume, error) {
	if name == "" {
		name = stringid.GenerateNonCryptoID()
	}

	v, err := daemon.volumes.Create(name, driverName, opts)
	if err != nil {
		if volumestore.IsNameConflict(err) {
			return nil, derr.ErrorVolumeNameTaken.WithArgs(name)
		}
		return nil, err
	}

	daemon.LogVolumeEvent(v.Name(), "create", map[string]string{"driver": v.DriverName()})
	return volumeToAPIType(v), nil
}
