package unit_tests

import (
	"testing"

	"go-task-wallet-service/api-gateway/internal/utils"
)

func TestConvertStringToInt_ValidDecimal(t *testing.T) {
	got, err := utils.ConvertStringToInt("42", 10, 16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got: %d", got)
	}
}

func TestConvertStringToInt_Negative(t *testing.T) {
	got, err := utils.ConvertStringToInt("-7", 10, 16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != -7 {
		t.Fatalf("expected -7, got: %d", got)
	}
}

func TestConvertStringToInt_Hex(t *testing.T) {
	got, err := utils.ConvertStringToInt("2a", 16, 16)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 42 {
		t.Fatalf("expected 42, got: %d", got)
	}
}

func TestConvertStringToInt_NonNumeric(t *testing.T) {
	got, err := utils.ConvertStringToInt("not-a-number", 10, 16)
	if err == nil {
		t.Fatalf("expected an error, got: %d", got)
	}
	if got != 0 {
		t.Fatalf("expected 0 on error, got: %d", got)
	}
}

func TestConvertStringToInt_Empty(t *testing.T) {
	got, err := utils.ConvertStringToInt("", 10, 16)
	if err == nil {
		t.Fatalf("expected an error, got: %d", got)
	}
	if got != 0 {
		t.Fatalf("expected 0 on error, got: %d", got)
	}
}

func TestConvertStringToInt_OverflowsBitSize(t *testing.T) {
	got, err := utils.ConvertStringToInt("999999", 10, 8)
	if err == nil {
		t.Fatalf("expected an overflow error for bitSize=8, got: %d", got)
	}
	if got != 0 {
		t.Fatalf("expected 0 on error, got: %d", got)
	}
}
