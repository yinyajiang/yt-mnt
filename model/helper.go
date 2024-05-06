package model

import (
	"database/sql/driver"
	"encoding/json"
)

func (f FormatList) Value() (driver.Value, error) {
	if len(f) == 0 {
		return nil, nil
	}
	return json.Marshal(f)
}

func (f *FormatList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, f)
}

func (m MediaEntryList) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

func (m *MediaEntryList) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	data, ok := value.([]byte)
	if !ok || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, m)
}

func (m *MediaEntry) SetNew(isnew bool) {
	m.IsNew = isnew
	for _, sub := range m.Entries {
		sub.IsNew = isnew
	}
}

func (m *MediaEntry) HasNew() bool {
	if m.IsNew {
		return true
	}
	for _, sub := range m.Entries {
		if sub.HasNew() {
			return true
		}
	}
	return false
}
