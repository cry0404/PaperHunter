package core

import (
	"fmt"
	"sync"

	"PaperHunter/internal/platform"
)

// Name 平台的唯一标识，例如："arxiv"、"acl"、"dblp"、"semantic"。
// New 构造具体平台实例的工厂函数；入参与出参严格使用 platform 包中的类型。
// DefaultConfig 返回该平台的一个可用默认配置（实现 platform.Config）。

type Provider struct {
	Name string

	New func(cfg platform.Config) (platform.Platform, error)

	DefaultConfig func() platform.Config
}

var (
	regMu    sync.RWMutex
	registry = map[string]Provider{}
)

func Register(p Provider) error {
	if p.Name == "" {
		return fmt.Errorf("provider 的名字不能为空")
	}
	if p.New == nil || p.DefaultConfig == nil {
		return fmt.Errorf("provider %s 的配置不正确", p.Name)
	}

	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := registry[p.Name]; exists {
		return fmt.Errorf("provider %s 已经注册过了", p.Name)
	}
	registry[p.Name] = p
	return nil
}


func MustRegister(p Provider) {
	if err := Register(p); err != nil {
		panic(err)
	}
}

func Get(name string) (Provider, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

/*
func List() []string {
	regMu.RLock()
	defer regMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}*/
