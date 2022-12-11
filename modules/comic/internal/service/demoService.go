package service

var DService = new(demoService)

type demoService struct{}

func (d *demoService) GetList() string {
	return "demoService"
}
