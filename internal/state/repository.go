package state

// Repository loads and persists on-disk agent state.
type Repository interface {
	Load(path string) (*AgentState, error)
	Save(path string, st *AgentState) error
}

// FileRepository reads and writes state.json on the local filesystem.
type FileRepository struct{}

func (FileRepository) Load(path string) (*AgentState, error) {
	return Load(path)
}

func (FileRepository) Save(path string, st *AgentState) error {
	return st.Save(path)
}
