package pg

import "testing"

func TestOrderStatusBuckets(t *testing.T) {
	if !isCancelledStatus("cancelled_by_client") {
		t.Fatal("cancelled_by_client must be cancelled")
	}
	if !isCancelledStatus("cancelled") {
		t.Fatal("legacy cancelled must be cancelled")
	}
	if isCancelledStatus("success") {
		t.Fatal("success must not be cancelled")
	}
	if !isSuccessStatus("success") {
		t.Fatal("success must be a sale")
	}
	if isSuccessStatus("created") {
		t.Fatal("created must not be a sale")
	}
}

func TestMoneyAndMarginCalculations(t *testing.T) {
	if got := parseMoneyCents("1 234,56"); got != 123456 {
		t.Fatalf("unexpected cents: %d", got)
	}
	if got := formatCents(-123456); got != "-1234.56" {
		t.Fatalf("unexpected money format: %s", got)
	}
	if got := percent(2500, 10000); got != 25 {
		t.Fatalf("unexpected percent: %v", got)
	}
	if got := percent(10, 0); got != 0 {
		t.Fatalf("zero denominator percent must be 0, got %v", got)
	}
}
