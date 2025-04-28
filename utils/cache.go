package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type CacheFile[V any] struct {
	Filename string
	Func     func() (*V, error)
}

func (c *CacheFile[V]) retrieveFromFile() (*V, error) {
	f, err := os.Open(c.Filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	val := new(V)
	if err := json.NewDecoder(f).Decode(val); err != nil {
		return nil, err
	}

	return val, nil
}

func (c *CacheFile[V]) saveToFile(val *V) error {
	f, err := os.OpenFile(c.Filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := json.NewEncoder(f).Encode(val); err != nil {
		return err
	}

	return nil
}

func (c *CacheFile[V]) Get() (*V, error) {
	val, err := c.retrieveFromFile()
	if err == nil {
		return val, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to retrieve from file: %w", err)
	}

	newVal, err := c.Func()
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve from function: %w", err)
	}

	if err := c.saveToFile(newVal); err != nil {
		return nil, fmt.Errorf("failed to save to file: %w", err)
	}

	return newVal, err
}

func NewCacheFile[V any](filename string, fn func() (*V, error)) *CacheFile[V] {
	return &CacheFile[V]{Filename: filename, Func: fn}
}
