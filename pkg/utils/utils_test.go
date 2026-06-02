package utils

import "testing"

/*
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
STACK TESTS
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
*/

func Test_Push(t *testing.T) {
	stack := Stack[int]{}
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	if stack.Size() != 3 {
		t.Errorf("Expected stack size 3, but got %d", stack.Size())
	}
}

func Test_Pop(t *testing.T) {
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

func Test_Peek(t *testing.T) {
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

func Test_IsEmpty(t *testing.T) {
	stack := Stack[int]{}

	if !stack.IsEmpty() {
		t.Errorf("Expected stack to be empty")
	}

	stack.Push(1)

	if stack.IsEmpty() {
		t.Errorf("Expected stack to not be empty")
	}
}

func Test_PopEmptyStack(t *testing.T) {
	stack := Stack[int]{}

	_, err := stack.Pop()
	if err == nil {
		t.Errorf("Expected error when popping from empty stack")
	}
}

func Test_PeekEmptyStack(t *testing.T) {
	stack := Stack[int]{}

	_, err := stack.Peek()
	if err == nil {
		t.Errorf("Expected error when peeking into empty stack")
	}
}

func Test_StringStack(t *testing.T) {
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

func Test_Size(t *testing.T) {
	stack := Stack[int]{}
	stack.Push(1)
	stack.Push(2)
	stack.Push(3)

	if stack.Size() != 3 {
		t.Errorf("Expected stack size 3, but got %d", stack.Size())
	}
}

/*
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
DICTIONARY TESTS
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
*/

// Test adding and retrieving values
func Test_Dictionary_AddAndGet(t *testing.T) {
	dict := &Dictionary[string]{}
	dict.Init(5)

	dict.Add("name", "John")
	dict.Delete("name")

	if _, exists := dict.Get("name"); exists {
		t.Error("Expected 'name' to be deleted")
	}
}

// Test deleting values
func Test_Dictionary_Delete(t *testing.T) {
	dict := Dictionary[string]{}
	dict.Init(5)

	dict.Add("name", "John")
	dict.Delete("name")

	if _, exists := dict.Get("name"); exists {
		t.Error("Expected 'name' to be deleted")
	}
}

/*
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
MDVR PROTOCOL HELPERS TESTS
~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
*/

func Test_VideoDescriptorStr(t *testing.T) {
	header := FilePacketHeader{
		DeviceId:         "1221223799",
		RequestStartTime: "20241114-170853",
		VideoLength:      "24",
		SerialNum:        "8",
		Channel:          "2",
	}

	expected := "1221223799_20241114-170853_24_8_2"
	result := VideoDescriptorStr(header)

	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func Test_ParseFilePacketHeader(t *testing.T) {
	input := "$FILE;V03;1221223799;1;1;20241114-170853;240;20241114-170810;60;4742224"
	var header FilePacketHeader
	err := ParseFilePacketHeader(input, &header)

	if err != nil {
		t.Errorf("Error should be nil, got %v", err)
	}
	if header.DeviceId != "1221223799" {
		t.Errorf("Expected DeviceId 1221223799, got %s", header.DeviceId)
	}
	if header.RequestStartTime != "20241114-170853" {
		t.Errorf("Expected RequestStartTime 20241114-170853, got %s", header.RequestStartTime)
	}
}

func Test_ParseFilePacketHeaderWithIncorrectPrefix(t *testing.T) {
	input := "$THISISWRONG;V03;1221223799;1;1;20241114-170853;240;20241114-170810;60;4742224"
	var header FilePacketHeader
	err := ParseFilePacketHeader(input, &header)

	if err == nil {
		t.Errorf("Error should not be nil after passing packet with incorrect prefix")
	}
}

func Test_ParseVideoPacketHeader(t *testing.T) {
	input := "$VIDEO;85000;all;0;20181127-091000;86400"
	var header VideoPacketHeader
	err := ParseVideoPacketHeader(input, &header)

	if err != nil {
		t.Errorf("Error should be nil, got %v", err)
	}
	if header.DeviceId != "85000" {
		t.Errorf("Expected DeviceId 85000, got %s", header.DeviceId)
	}
	if header.RequestStartTime != "20181127-091000" {
		t.Errorf("Expected RequestStartTime 20181127-091000, got %s", header.RequestStartTime)
	}
}

func Test_ParseVideoPacketHeaderWithIncorrectPrefix(t *testing.T) {
	input := "$THISISWRONG;85000;all;0;20181127-091000;86400"
	var header VideoPacketHeader
	err := ParseVideoPacketHeader(input, &header)

	if err == nil {
		t.Errorf("Error should not be nil after passing packet with incorrect prefix")
	}
}

func Test_ParseVideoPacketHeaderWithEmptyString(t *testing.T) {
	input := "$THISISWRONG;85000;all;0;20181127-091000;86400"
	var header VideoPacketHeader
	err := ParseVideoPacketHeader(input, &header)

	if err == nil {
		t.Errorf("Error should not be nil after passing packet with incorrect prefix")
	}
}

func Test_ParseReqMatchStringFromVideoPacketHeader(t *testing.T) {
	header := VideoPacketHeader{
		DeviceId:         "1221223799",
		RequestStartTime: "20241114-170853",
		RequestLength:    "240",
	}

	expected := "122122379920241114-170853240"
	result, err := ParseReqMatchStringFromVideoPacketHeader(&header)

	if err != nil {
		t.Errorf("Error should be nil, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func Test_ParseReqMatchStringFromVideoDescription(t *testing.T) {
	desc := VideoDescription{
		DeviceId:         "1221223799",
		RequestStartTime: "20241114-170853",
		RequestLength:    "240",
	}

	expected := "122122379920241114-170853240"
	result, err := ParseReqMatchStringFromVideoDescription(&desc)

	if err != nil {
		t.Errorf("Error should be nil, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
