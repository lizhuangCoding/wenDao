package chatcore

import (
	"fmt"
	"strings"
	"sync"
)

type PluginRegistry interface {
	Register(plugin AgentPlugin, options ...RegisterOption) error
	Get(name string) (AgentPlugin, bool)
	Default() (AgentPlugin, bool)
	List() []PluginManifest
}

type RegisterOption func(*registerOptions)

type registerOptions struct {
	defaultPlugin bool
}

func WithDefaultPlugin() RegisterOption {
	return func(options *registerOptions) {
		options.defaultPlugin = true
	}
}

type pluginRegistry struct {
	mu          sync.RWMutex
	plugins     map[string]AgentPlugin
	defaultName string
}

func NewPluginRegistry() PluginRegistry {
	return &pluginRegistry{plugins: make(map[string]AgentPlugin)}
}

func (r *pluginRegistry) Register(plugin AgentPlugin, options ...RegisterOption) error {
	if plugin == nil {
		return fmt.Errorf("plugin is required")
	}
	manifest := plugin.Manifest()
	name := strings.TrimSpace(manifest.Name)
	if name == "" {
		return fmt.Errorf("plugin manifest name is required")
	}
	config := registerOptions{}
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[name] = plugin
	if config.defaultPlugin || r.defaultName == "" {
		r.defaultName = name
	}
	return nil
}

func (r *pluginRegistry) Get(name string) (AgentPlugin, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[strings.TrimSpace(name)]
	return plugin, ok
}

func (r *pluginRegistry) Default() (AgentPlugin, bool) {
	if r == nil {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	plugin, ok := r.plugins[r.defaultName]
	return plugin, ok
}

func (r *pluginRegistry) List() []PluginManifest {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	manifests := make([]PluginManifest, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		manifests = append(manifests, plugin.Manifest())
	}
	return manifests
}
