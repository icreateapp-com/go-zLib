package z

import (
	"errors"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	entranslations "github.com/go-playground/validator/v10/translations/en"
	"log"
	"reflect"
	"strings"
)

var (
	uni   *ut.UniversalTranslator
	trans ut.Translator
)

type valid struct {
}

var Validator valid

// Init 初始化翻译器
func (valid *valid) Init() {
	zhTrans := zh.New()
	enTrans := en.New()
	uni = ut.New(enTrans, zhTrans)
	trans, _ = uni.GetTranslator("en")
	validate := binding.Validator.Engine().(*validator.Validate)
	if err := entranslations.RegisterDefaultTranslations(validate, trans); err != nil {
		log.Fatalln(err)
	}
}

// T 翻译错误
func (valid *valid) T(err error, req interface{}) string {

	labels := map[string]string{}

	t := reflect.TypeOf(req)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		label := field.Tag.Get("label")
		if len(label) == 0 {
			label = field.Tag.Get("json")
		}
		labels[field.Name] = label
	}

	var errs validator.ValidationErrors

	if errors.As(err, &errs) {
		if len(errs) > 0 {
			err := errs[0].Translate(trans)
			field := errs[0].Field()
			label := labels[field]

			if len(label) < 1 {
				label = field
			}

			return strings.Replace(err, field, label, -1)
		}
	}

	return err.Error()
}
