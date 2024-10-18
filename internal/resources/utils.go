package resources

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"k8s.io/apimachinery/pkg/api/equality"
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FormatResource(obj interface{}) string {
	jsonData, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v\n", obj)
	}

	return string(jsonData)
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
	specBytes, err := json.Marshal(obj)

	if err != nil {
		return ""
	}

	hash := sha256.Sum256(specBytes)
	return fmt.Sprintf("%x", hash[:])
}

func HasSpecChanged(obj1 client.Object, obj2 client.Object) bool {
	spec1, ok1 := ExtractSpec(obj1)
	spec2, ok2 := ExtractSpec(obj2)

	if ok1 && ok2 {
		return !equality.Semantic.DeepEqual(spec1, spec2)
	}

	return !equality.Semantic.DeepEqual(obj1, obj2)
}
