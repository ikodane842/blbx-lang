package interpreter

import "sort"

// Sorted keys keep indexes and values aligned across separate property reads.
func objectEntries(object Value, property string) Value {
	keys := make([]string, 0, len(object.Object))
	for key := range object.Object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]Value, 0, len(keys))
	for _, key := range keys {
		if property == "indexes" {
			values = append(values, String(key))
		} else {
			values = append(values, bindReceiver(object.Object[key], object))
		}
	}
	return Array(values)
}
