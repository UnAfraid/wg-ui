package subscription

import (
	"encoding/json"

	"github.com/UnAfraid/wg-ui/api/model"
)

func notify[T model.NodeChangedEvent](bytes []byte, observerChan chan<- T) error {
	var node T
	if err := json.Unmarshal(bytes, &node); err != nil {
		return err
	}
	observerChan <- node
	return nil
}
