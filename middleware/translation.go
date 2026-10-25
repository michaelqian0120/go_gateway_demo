package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	"github.com/go-playground/universal-translator"
	"go_gateway_demo/dao"
	"go_gateway_demo/dto"
	"go_gateway_demo/public"
	"golang_common/lib"
	"gopkg.in/go-playground/validator.v9"
	en_translations "gopkg.in/go-playground/validator.v9/translations/en"
	zh_translations "gopkg.in/go-playground/validator.v9/translations/zh"
	"reflect"
	"regexp"
	"strings"
)

// 设置Translation
func TranslationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		//参照：https://github.com/go-playground/validator/blob/v9/_examples/translations/main.go

		//设置支持语言
		en := en.New()
		zh := zh.New()

		//设置国际化翻译器
		uni := ut.New(zh, zh, en)
		val := validator.New()

		//根据参数取翻译器实例
		locale := c.DefaultQuery("locale", "zh")
		trans, _ := uni.GetTranslator(locale)

		//翻译器注册到validator
		switch locale {
		case "en":
			en_translations.RegisterDefaultTranslations(val, trans)
			val.RegisterTagNameFunc(func(fld reflect.StructField) string {
				return fld.Tag.Get("en_comment")
			})
			break
		default:
			zh_translations.RegisterDefaultTranslations(val, trans)
			val.RegisterTagNameFunc(func(fld reflect.StructField) string {
				return fld.Tag.Get("comment")
			})

			//自定义验证方法
			//https://github.com/go-playground/validator/blob/v9/_examples/custom-validation/main.go
			val.RegisterValidation("is-validuser", func(fl validator.FieldLevel) bool {
				return fl.Field().String() == "admin"
			})

			//自定义验证器
			//https://github.com/go-playground/validator/blob/v9/_examples/translations/main.go
			val.RegisterTranslation("is-validuser", trans, func(ut ut.Translator) error {
				return ut.Add("is-validuser", "{0} 填写不正确哦", true)
			}, func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T("is-validuser", fe.Field())
				return t
			})

			//服务名称校验：6-128位字母、数字、下划线
			val.RegisterValidation("service_name", func(fl validator.FieldLevel) bool {
				matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]{6,128}$`, fl.Field().String())
				return matched
			})
			val.RegisterTranslation("service_name", trans, func(ut ut.Translator) error {
				return ut.Add("service_name", "{0} 必须是6-128位字母、数字、下划线", true)
			}, func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T("service_name", fe.Field())
				return t
			})

			//服务名称唯一性校验
			val.RegisterValidation("unique_service_name", func(fl validator.FieldLevel) bool {
				tx, err := lib.GetGormPool("default")
				if err != nil {
					return false
				}
				existServiceInfo, err := (&dao.ServiceInfo{}).FindByName(tx, fl.Field().String())
				if err != nil {
					return false
				}
				return existServiceInfo.Id == 0
			})
			val.RegisterTranslation("unique_service_name", trans, func(ut ut.Translator) error {
				return ut.Add("unique_service_name", "{0} 已存在", true)
			}, func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T("unique_service_name", fe.Field())
				return t
			})

			//ip列表与权重列表数量需保持一致
			val.RegisterStructValidation(func(sl validator.StructLevel) {
				input := sl.Current().Interface().(dto.ServiceAddHTTPInput)
				if len(strings.Split(input.IpList, ",")) != len(strings.Split(input.WeightList, ",")) {
					sl.ReportError(input.WeightList, "WeightList", "WeightList", "ip_weight_count", "")
				}
			}, dto.ServiceAddHTTPInput{})
			val.RegisterTranslation("ip_weight_count", trans, func(ut ut.Translator) error {
				return ut.Add("ip_weight_count", "ip列表与权重列表数量不一致", true)
			}, func(ut ut.Translator, fe validator.FieldError) string {
				t, _ := ut.T("ip_weight_count")
				return t
			})
			break
		}
		c.Set(public.TranslatorKey, trans)
		c.Set(public.ValidatorKey, val)
		c.Next()
	}
}
