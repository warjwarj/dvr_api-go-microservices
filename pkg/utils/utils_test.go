package utils

import "testing"

func TestPush(t *testing.T) {
	stack := Stack[int]{}
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	if stack.Size() != 3 {
		t.Errorf("Expected stack size 3, but got %d", stack.Size())
	}
}

func TestPop(t *testing.T) {
	stack := Stack[int]{}
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	item, err := stack.Pop()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if item != 3 {
		t.Errorf("Expected 3, but got %d", item)
	}

	if stack.Size() != 2 {
		t.Errorf("Expected stack size 2, but got %d", stack.Size())
	}

	item, err = stack.Pop()
	item, err = stack.Pop()
	item, err = stack.Pop()
}

func TestPeek(t *testing.T) {
	stack := Stack[int]{}
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	item, err := stack.Peek()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if item != 3 {
		t.Errorf("Expected 3, but got %d", item)
	}

	if stack.Size() != 3 {
		t.Errorf("Expected stack size 3, but got %d", stack.Size())
	}
}

func TestIsEmpty(t *testing.T) {
	stack := Stack[int]{}

	if !stack.IsEmpty() {
		t.Errorf("Expected stack to be empty")
	}

	stack.Push(1)

	if stack.IsEmpty() {
		t.Errorf("Expected stack to not be empty")
	}
}

func TestPopEmptyStack(t *testing.T) {
	stack := Stack[int]{}

	_, err := stack.Pop()
	if err == nil {
		t.Errorf("Expected error when popping from empty stack")
	}
}

func TestPeekEmptyStack(t *testing.T) {
	stack := Stack[int]{}

	_, err := stack.Peek()
	if err == nil {
		t.Errorf("Expected error when peeking into empty stack")
	}
}

func TestStringStack(t *testing.T) {
	stack := Stack[string]{}
	stack.Push("hello")
	stack.Push("world")

	item, err := stack.Pop()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if item != "world" {
		t.Errorf("Expected 'world', but got %s", item)
	}

	item, err = stack.Peek()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if item != "hello" {
		t.Errorf("Expected 'hello', but got %s", item)
	}
}

func TestSize(t *testing.T) {
	stack := Stack[int]{}
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	if stack.Size() != 3 {
		t.Errorf("Expected stack size 3, but got %d", stack.Size())
	}
}

// Test adding and retrieving values
func TestDictionary_AddAndGet(t *testing.T) {
	dict := &Dictionary[string]{}
	dict.Init(5)

	dict.Add("name", "John")
	dict.Delete("name")

	if _, exists := dict.Get("name"); exists {
		t.Error("Expected 'name' to be deleted")
	}
}

// Test deleting values
func TestDictionary_Delete(t *testing.T) {
	dict := Dictionary[string]{}
	dict.Init(5)

	dict.Add("name", "John")
	dict.Delete("name")

	if _, exists := dict.Get("name"); exists {
		t.Error("Expected 'name' to be deleted")
	}
}
