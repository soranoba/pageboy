package magion

import "errors"

type Paging struct {
	Page    uint
	PerPage uint
}

func NewPaging(page uint, perPage uint) *Paging {
	return &Paging{
		Page:    page,
		PerPage: perPage,
	}
}

func (paging *Paging) Validate() error {
	if paging.PerPage == 0 {
		return errors.New("PerPage parameter must be greater than 0")
	}
	return nil
}
