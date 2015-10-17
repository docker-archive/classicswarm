package casttype

import (
	"encoding/json"
)

type MyUint64Type uint64

type Bytes []byte

func (this Bytes) MarshalJSON() ([]byte, error) {
	return json.Marshal([]byte(this))
}

func (this *Bytes) UnmarshalJSON(data []byte) error {
	v := new([]byte)
	err := json.Unmarshal(data, v)
	if err != nil {
		return err
	}
	*this = *v
	return nil
}
