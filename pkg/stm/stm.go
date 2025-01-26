package stm

import "fmt"

type Token[T any] struct {
	Value T
}

type State[T any] struct {
	Name  string
	First bool
	Run   func(value T, stm *StateMachine[T]) error
}

type StateMachine[T any] struct {
	tokens       []Token[T]
	states       map[string]State[T]
	currentState string
	position     int
}

func (stm *StateMachine[T]) Init(values []T) *StateMachine[T] {
	stm.tokens = make([]Token[T], len(values))
	for i, value := range values {
		stm.tokens[i] = Token[T]{Value: value}
	}
	stm.states = make(map[string]State[T])
	return stm
}

func (stm *StateMachine[T]) AddState(state State[T]) *StateMachine[T] {
	stm.states[state.Name] = state
	if state.First {
		stm.currentState = state.Name
	}
	return stm
}

func (stm *StateMachine[T]) Parse() error {
	for stm.position < len(stm.tokens) {
		current, ok := stm.states[stm.currentState]
		if !ok {
			return fmt.Errorf("[%s] state not found", stm.currentState)
		}

		fmt.Printf("[%s] state goes next\n", stm.currentState)
		err := current.Run(stm.tokens[stm.position].Value, stm)
		if err != nil {
			return err
		}
	}

	fmt.Println("all tokens parsed")

	return nil
}

func (stm *StateMachine[T]) Next(name string) error {
	_, ok := stm.states[name]
	if !ok {
		return fmt.Errorf("[%v] state not found", name)
	}

	stm.currentState = name
	stm.position += 1
	return nil
}

func (stm *StateMachine[T]) Token(delta int) (*Token[T], error) {
	pos := stm.position + delta
	if pos < 0 || pos >= len(stm.tokens) {
		return nil, fmt.Errorf("%d is out bounds", stm.position+delta)
	}

	return &stm.tokens[pos], nil
}

func (stm *StateMachine[T]) Consume(count int) {
	stm.position += count
}
