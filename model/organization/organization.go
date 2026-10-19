// Package organization owns the public identity of the student representation.
package organization

type Organization struct{ Name string }

type Service struct{ organization Organization }

func New(name string) *Service { return &Service{organization: Organization{Name: name}} }

func (s *Service) Public() Organization { return s.organization }
