package oauth2

import (
	stdContext "context"

	"github.com/kataras/iris/v14/core/pkg/cache"
)

// DeviceCodeStore defines methods to persist device codes and retrieve them by code or user code.
type DeviceCodeStore interface {
	// InsertDeviceCode saves a device code entry in the store.
	InsertDeviceCode(ctx stdContext.Context, device *DeviceCode) error
	// GetAndDeleteDeviceCode retrieves and deletes a device code by its code value.
	GetAndDeleteDeviceCode(ctx stdContext.Context, code string) (*DeviceCode, bool, error)
	// GetDeviceCodeByUserCode retrieves a device code using the user code field.
	GetDeviceCodeByUserCode(ctx stdContext.Context, userCode string) (*DeviceCode, bool, error)
}

// memDeviceCodeStore implements DeviceCodeStore using an in-memory cache.
type memDeviceCodeStore struct {
	// device is a cache that can iterate and store device codes.
	device cache.CacheIterator[DeviceCode]
}

// NewMemoryDeviceCodeStore creates an in-memory device code store.
func NewMemoryDeviceCodeStore() DeviceCodeStore {
	return &memDeviceCodeStore{
		device: cache.NewMemoryCache[DeviceCode](0),
	}
}

// InsertDeviceCode adds a new device code to the in-memory cache.
func (s *memDeviceCodeStore) InsertDeviceCode(ctx stdContext.Context, device *DeviceCode) error {
	return s.device.Set(ctx, device.Code, device)
}

// GetAndDeleteDeviceCode retrieves a device code by its code and removes it from the cache.
func (s *memDeviceCodeStore) GetAndDeleteDeviceCode(ctx stdContext.Context, code string) (*DeviceCode, bool, error) {
	deviceCode, ok, err := s.device.Get(ctx, code)
	if ok {
		err = s.device.Delete(ctx, code)
	}
	return deviceCode, ok, err
}

// GetDeviceCodeByUserCode finds a device code matching the provided userCode in the cache.
func (s *memDeviceCodeStore) GetDeviceCodeByUserCode(ctx stdContext.Context, userCode string) (*DeviceCode, bool, error) {
	_, value, found, err := s.device.First(ctx, func(v *DeviceCode) bool {
		return v.UserCode == userCode
	})

	return value, found, err
}
