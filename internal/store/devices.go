package store

import (
	"strings"
	"time"

	"tinypanel-hub/internal/domain"
)

const bindCodeTTL = 10 * time.Minute

func (s *FileStore) DeviceByCredentials(deviceID, secretHash string) (domain.Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.state.data.Devices {
		if device.ID == deviceID && device.SecretHash == secretHash {
			return device, true
		}
	}
	return domain.Device{}, false
}

func (s *FileStore) Device(ownerID, deviceID string) (domain.Device, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, device := range s.state.data.Devices {
		if device.ID == deviceID && device.OwnerID == ownerID {
			return device, true
		}
	}
	return domain.Device{}, false
}

func (s *FileStore) Devices(ownerID string) []domain.Device {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := []domain.Device{}
	for _, device := range s.state.data.Devices {
		if device.OwnerID == ownerID {
			out = append(out, device)
		}
	}
	return out
}

func (s *FileStore) HelloDevice(deviceID, secretHash string) (domain.DeviceHello, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for i := range s.state.data.Devices {
		device := &s.state.data.Devices[i]
		if device.ID != deviceID {
			continue
		}
		if device.SecretHash != secretHash {
			return domain.DeviceHello{}, errInvalidDeviceSecret
		}
		device.LastSeenAt = &now
		if device.OwnerID == "" {
			if device.BindCode == "" || device.BindCodeExpiresAt == nil || now.After(*device.BindCodeExpiresAt) {
				expires := now.Add(bindCodeTTL)
				device.BindCode = s.uniqueBindCode()
				device.BindCodeExpiresAt = &expires
			}
		}
		if err := s.state.save(); err != nil {
			return domain.DeviceHello{}, err
		}
		return deviceHello(*device, now), nil
	}

	expires := now.Add(bindCodeTTL)
	device := domain.Device{
		ID:                deviceID,
		SecretHash:        secretHash,
		BindCode:          s.uniqueBindCode(),
		BindCodeExpiresAt: &expires,
		LastSeenAt:        &now,
		CreatedAt:         now,
	}
	s.state.data.Devices = append(s.state.data.Devices, device)
	if err := s.state.save(); err != nil {
		return domain.DeviceHello{}, err
	}
	return deviceHello(device, now), nil
}

func (s *FileStore) BindDevice(ownerID, bindCode, name string) (domain.Device, bool, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	for i := range s.state.data.Devices {
		device := &s.state.data.Devices[i]
		if device.BindCode != bindCode {
			continue
		}
		if device.OwnerID != "" || device.BindCodeExpiresAt == nil || now.After(*device.BindCodeExpiresAt) {
			return domain.Device{}, true, false, nil
		}
		device.OwnerID = ownerID
		device.Name = strings.TrimSpace(name)
		if device.Name == "" {
			device.Name = device.ID
		}
		device.BindCode = ""
		device.BindCodeExpiresAt = nil
		device.BoundAt = &now
		if err := s.state.save(); err != nil {
			return domain.Device{}, true, false, err
		}
		return *device, true, true, nil
	}
	return domain.Device{}, false, false, nil
}

func (s *FileStore) UpdateDevice(ownerID, deviceID, name string) (domain.Device, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.state.data.Devices {
		device := &s.state.data.Devices[i]
		if device.ID != deviceID || device.OwnerID != ownerID {
			continue
		}
		device.Name = strings.TrimSpace(name)
		if err := s.state.save(); err != nil {
			return domain.Device{}, true, err
		}
		return *device, true, nil
	}
	return domain.Device{}, false, nil
}

func (s *FileStore) DeleteDevice(ownerID, deviceID string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, device := range s.state.data.Devices {
		if device.ID != deviceID || device.OwnerID != ownerID {
			continue
		}
		s.state.data.Devices = append(s.state.data.Devices[:i], s.state.data.Devices[i+1:]...)
		if err := s.state.save(); err != nil {
			return true, err
		}
		return true, nil
	}
	return false, nil
}

func deviceHello(device domain.Device, now time.Time) domain.DeviceHello {
	hello := domain.DeviceHello{
		DeviceID:    device.ID,
		Bound:       device.OwnerID != "",
		Name:        device.Name,
		BindCode:    device.BindCode,
		BindCodeTTL: int(bindCodeTTL.Seconds()),
		ServerTime:  now,
		BoundAt:     device.BoundAt,
	}
	if hello.Bound {
		hello.BindCode = ""
		hello.BindCodeTTL = 0
	}
	return hello
}

func (s *FileStore) uniqueBindCode() string {
	for {
		code := randomDigits(6)
		if !s.bindCodeExists(code) {
			return code
		}
	}
}

func (s *FileStore) bindCodeExists(code string) bool {
	for _, device := range s.state.data.Devices {
		if device.BindCode == code {
			return true
		}
	}
	return false
}
