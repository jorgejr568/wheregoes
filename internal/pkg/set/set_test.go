package set

import (
	"reflect"
	"sort"
	"testing"
)

func TestSet_Add(t *testing.T) {
	s := New[string]()
	
	s.Add("item1")
	if !s.Contains("item1") {
		t.Errorf("expected set to contain 'item1'")
	}
	
	if s.Len() != 1 {
		t.Errorf("expected set length to be 1, got %d", s.Len())
	}
	
	s.Add("item1")
	if s.Len() != 1 {
		t.Errorf("expected set length to remain 1 after adding duplicate, got %d", s.Len())
	}
}

func TestSet_Remove(t *testing.T) {
	s := New[string]()
	
	s.Add("item1")
	s.Add("item2")
	
	s.Remove("item1")
	if s.Contains("item1") {
		t.Errorf("expected 'item1' to be removed from set")
	}
	
	if !s.Contains("item2") {
		t.Errorf("expected 'item2' to still be in set")
	}
	
	if s.Len() != 1 {
		t.Errorf("expected set length to be 1, got %d", s.Len())
	}
	
	s.Remove("nonexistent")
	if s.Len() != 1 {
		t.Errorf("expected set length to remain 1 after removing nonexistent item, got %d", s.Len())
	}
}

func TestSet_Contains(t *testing.T) {
	s := New[int]()
	
	if s.Contains(1) {
		t.Errorf("expected empty set to not contain 1")
	}
	
	s.Add(1)
	s.Add(2)
	
	if !s.Contains(1) {
		t.Errorf("expected set to contain 1")
	}
	
	if !s.Contains(2) {
		t.Errorf("expected set to contain 2")
	}
	
	if s.Contains(3) {
		t.Errorf("expected set to not contain 3")
	}
}

func TestSet_Len(t *testing.T) {
	s := New[string]()
	
	if s.Len() != 0 {
		t.Errorf("expected empty set length to be 0, got %d", s.Len())
	}
	
	s.Add("item1")
	if s.Len() != 1 {
		t.Errorf("expected set length to be 1, got %d", s.Len())
	}
	
	s.Add("item2")
	if s.Len() != 2 {
		t.Errorf("expected set length to be 2, got %d", s.Len())
	}
	
	s.Add("item1")
	if s.Len() != 2 {
		t.Errorf("expected set length to remain 2 after adding duplicate, got %d", s.Len())
	}
	
	s.Remove("item1")
	if s.Len() != 1 {
		t.Errorf("expected set length to be 1 after removal, got %d", s.Len())
	}
}

func TestSet_Values(t *testing.T) {
	s := New[string]()
	
	values := s.Values()
	if len(values) != 0 {
		t.Errorf("expected empty set values to have length 0, got %d", len(values))
	}
	
	s.Add("item1")
	s.Add("item2")
	s.Add("item3")
	
	values = s.Values()
	if len(values) != 3 {
		t.Errorf("expected values length to be 3, got %d", len(values))
	}
	
	sort.Strings(values)
	expected := []string{"item1", "item2", "item3"}
	if !reflect.DeepEqual(values, expected) {
		t.Errorf("expected values to be %v, got %v", expected, values)
	}
}

func TestSet_IntegerType(t *testing.T) {
	s := New[int]()
	
	s.Add(1)
	s.Add(2)
	s.Add(3)
	
	if s.Len() != 3 {
		t.Errorf("expected integer set length to be 3, got %d", s.Len())
	}
	
	if !s.Contains(2) {
		t.Errorf("expected integer set to contain 2")
	}
	
	s.Remove(2)
	if s.Contains(2) {
		t.Errorf("expected 2 to be removed from integer set")
	}
}

func TestSet_EmptyOperations(t *testing.T) {
	s := New[string]()
	
	s.Remove("nonexistent")
	if s.Len() != 0 {
		t.Errorf("expected empty set to remain empty after removing nonexistent item")
	}
	
	values := s.Values()
	if len(values) != 0 {
		t.Errorf("expected empty set values to be empty slice")
	}
	
	if values == nil {
		t.Errorf("expected Values() to return empty slice, not nil")
	}
}

func TestNew(t *testing.T) {
	s := New[string]()
	
	if s == nil {
		t.Errorf("expected New() to return non-nil set")
	}
	
	if s.Len() != 0 {
		t.Errorf("expected new set to be empty")
	}
}