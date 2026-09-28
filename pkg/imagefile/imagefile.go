// Package imagefile проверяет картинки перед отправкой в хранилище.
//
// Валидация живёт отдельным пакетом, а не в обработчике загрузки, потому что
// файл приезжает тремя путями: /upload, /upload/bulk и полем обложки в формах
// вишлиста и подарка. Проверка только в первом из них обходится сменой поля.
package imagefile

import (
	"fmt"
	"net/http"
)

// MaxSize — предел, обещанный пользователю в интерфейсе загрузки.
const MaxSize = 10 << 20 // 10 МБ

// allowed — типы, которые умеет показать фронт. Определяются по сигнатуре
// файла, а не по заголовку из запроса: заголовок присылает клиент.
var allowed = map[string]string{
	"image/jpeg": "JPG",
	"image/png":  "PNG",
	"image/webp": "WEBP",
}

// Validate возвращает понятную пользователю ошибку, если файл не картинка
// поддерживаемого формата или тяжелее предела.
func Validate(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("файл пустой")
	}
	if len(data) > MaxSize {
		return fmt.Errorf("файл больше %d МБ — сожмите его или выберите другой", MaxSize>>20)
	}
	contentType := http.DetectContentType(data)
	if _, ok := allowed[contentType]; !ok {
		return fmt.Errorf("формат не поддерживается — нужен JPG, PNG или WEBP")
	}
	return nil
}
