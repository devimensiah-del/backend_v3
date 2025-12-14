package analysisbysteps

// Service handles business logic for analysis steps
// TODO: Implement service methods as needed
type Service struct {
	repo *Repository
}

// NewService creates a new Service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}
