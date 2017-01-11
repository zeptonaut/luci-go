// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package config

import (
	"net/url"

	"github.com/luci/luci-go/common/errors"
	log "github.com/luci/luci-go/common/logging"
	"github.com/luci/luci-go/logdog/api/config/svcconfig"
	"github.com/luci/luci-go/luci_config/common/cfgtypes"
	"github.com/luci/luci-go/luci_config/server/cfgclient"
	"github.com/luci/luci-go/luci_config/server/cfgclient/textproto"

	"golang.org/x/net/context"
)

var (
	// ErrInvalidConfig is returned when the configuration exists, but is invalid.
	ErrInvalidConfig = errors.New("invalid configuration")
)

// Config is the LogDog Coordinator service configuration.
type Config struct {
	svcconfig.Config
	// Settings are per-instance settings.
	Settings Settings

	// ConfigServiceURL is the config service's URL.
	ConfigServiceURL url.URL `json:"-"`
	// ConfigSet is the name of the service config set that is being used.
	ConfigSet cfgtypes.ConfigSet `json:"-"`
	// ServiceConfigPath is the path within ConfigSet of the service
	// configuration.
	ServiceConfigPath string `json:"-"`
}

// ServiceConfigPath returns the config set and path for this application's
// service configuration.
func ServiceConfigPath(c context.Context) (cfgtypes.ConfigSet, string) {
	return cfgclient.CurrentServiceConfigSet(c), svcconfig.ServiceConfigFilename
}

// Load loads the service configuration. This includes:
//	- The config service settings.
//	- The service configuration, loaded from the config service.
//	- Additional Settings data from datastore via settings.
//
// The service config is minimally validated prior to being returned.
func Load(c context.Context) (*Config, error) {
	// Unmarshal the config into service configuration.
	cfg := Config{
		ConfigServiceURL: cfgclient.ServiceURL(c),
	}
	cfg.ConfigSet, cfg.ServiceConfigPath = ServiceConfigPath(c)

	// Load our service-level config.
	if err := cfgclient.Get(c, cfgclient.AsService, cfg.ConfigSet, cfg.ServiceConfigPath,
		textproto.Message(&cfg.Config), nil); err != nil {

		log.Fields{
			log.ErrorKey: err,
			"configSet":  cfg.ConfigSet,
			"configPath": cfg.ServiceConfigPath,
		}.Errorf(c, "Failed to load configuration from config service.")
		return nil, err
	}

	// Validate the configuration.
	if err := validateServiceConfig(&cfg.Config); err != nil {
		log.WithError(err).Errorf(c, "Invalid Coordinator configuration.")
		return nil, ErrInvalidConfig
	}

	// Load our settings.
	if err := cfg.Settings.Load(c); err != nil {
		log.WithError(err).Errorf(c, "Failed to load settings.")
		return nil, ErrInvalidConfig
	}

	return &cfg, nil
}

// validateServiceConfig checks the supplied service config object to ensure
// that it meets a minimum configuration standard expected by our endpoitns and
// handlers.
func validateServiceConfig(cc *svcconfig.Config) error {
	switch {
	case cc == nil:
		return errors.New("configuration is nil")
	case cc.GetCoordinator() == nil:
		return errors.New("no Coordinator configuration")
	default:
		return nil
	}
}
