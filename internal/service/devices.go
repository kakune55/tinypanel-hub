package service

import "tinypanel-hub/internal/domain"

type DeviceService struct {
	store DeviceStore
}

func (s DeviceService) ByCredentials(deviceID, secretHash string) (domain.Device, bool) {
	return s.store.DeviceByCredentials(deviceID, secretHash)
}

func (s DeviceService) Hello(deviceID, secretHash string) (domain.DeviceHello, error) {
	return s.store.HelloDevice(deviceID, secretHash)
}

func (s DeviceService) List(ownerID string) []domain.Device {
	return s.store.Devices(ownerID)
}

func (s DeviceService) Get(ownerID, deviceID string) (domain.Device, bool) {
	return s.store.Device(ownerID, deviceID)
}

func (s DeviceService) Bind(ownerID, bindCode, name string) (domain.Device, bool, bool, error) {
	return s.store.BindDevice(ownerID, bindCode, name)
}

func (s DeviceService) Update(ownerID, deviceID, name string) (domain.Device, bool, error) {
	return s.store.UpdateDevice(ownerID, deviceID, name)
}

func (s DeviceService) Delete(ownerID, deviceID string) (bool, error) {
	return s.store.DeleteDevice(ownerID, deviceID)
}
