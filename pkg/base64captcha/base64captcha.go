package base64captcha

import (
	"errors"
	"github.com/mojocn/base64Captcha"
)

type Captcha struct {
	driver base64Captcha.Driver
	store  base64Captcha.Store
}

func (c *Captcha) Generate() (string, error) {
	id, _, answer := c.driver.GenerateIdQuestionAnswer()
	c.store.Set(id, answer)
	return id, nil
}

func (c *Captcha) GetImageData(id string) (string, error) {
	content := c.store.Get(id, false)
	if content == "" {
		return "", errors.New("id not found")
	}
	item, err := c.driver.DrawCaptcha(content)
	if err != nil {
		return "", err
	}
	return item.EncodeB64string(), nil
}

func (c *Captcha) Verify(id, answer string) bool {
	return c.store.Verify(id, answer, true)
}

func NewCaptcha(driver base64Captcha.Driver, store base64Captcha.Store) *Captcha {
	return &Captcha{
		driver: driver,
		store:  store,
	}
}
