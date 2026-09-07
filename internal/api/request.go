package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/gofiber/fiber/v2"
)

func parseRequest(ctx *fiber.Ctx, request any) error {
	if !ctx.Is("json") {
		return errors.New("JSON content type is required")
	}
	decoder := json.NewDecoder(bytes.NewReader(ctx.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(request); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return errors.New("request must contain one JSON value")
	}
	return nil
}
