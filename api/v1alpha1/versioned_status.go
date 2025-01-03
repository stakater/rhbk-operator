package v1alpha1

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
)

type VersionedStatus struct {
	ResourceVersions map[string]string `json:"resourceVersions,omitempty"`
}

func (s *VersionedStatus) UpdateVersion(key string, obj interface{}) {
	if s.ResourceVersions == nil {
		s.ResourceVersions = make(map[string]string)
	}

	s.ResourceVersions[key] = ComputeHash(obj)
}

func (s *VersionedStatus) HasBeenUpdated(key string, obj interface{}) bool {
	current, ok := s.ResourceVersions[key]
	return ok && current == ComputeHash(obj)
}

func ExtractSpec(obj interface{}) (interface{}, bool) {
	v := reflect.ValueOf(obj)

	// Handle pointers by dereferencing them
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	// Ensure the object is a struct
	if v.Kind() != reflect.Struct {
		return nil, false
	}

	// Attempt to get the "Spec" field (Go fields are case-sensitive)
	field := v.FieldByName("Spec")
	if field.IsValid() {
		// Return the value of the "Spec" field and true
		return field.Interface(), true
	}
	return nil, false
}

func ComputeHash(obj interface{}) string {
	var hashable interface{}
	spec, ok := ExtractSpec(obj)
	if ok {
		hashable = spec
	} else {
		hashable = obj
	}

	var objString []byte
	specBytes, err := json.Marshal(hashable)

	if err != nil {
		objString = []byte(fmt.Sprintf("%v", hashable))
	} else {
		objString = specBytes
	}

	hash := md5.Sum(objString)
	return base64.StdEncoding.EncodeToString(hash[:])
}
