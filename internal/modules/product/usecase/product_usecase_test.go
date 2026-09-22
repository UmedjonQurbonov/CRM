package usecase

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockProductRepo struct {
	byID     map[uuid.UUID]*domain.Product
	byQRCode map[string]*domain.Product
	bySKU    map[string]*domain.Product
}

func newMockProductRepo() *mockProductRepo {
	return &mockProductRepo{
		byID:     make(map[uuid.UUID]*domain.Product),
		byQRCode: make(map[string]*domain.Product),
		bySKU:    make(map[string]*domain.Product),
	}
}

func (m *mockProductRepo) Create(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	if _, exists := m.bySKU[p.SKU]; exists {
		return nil, domain.ErrDuplicateSKU
	}
	if _, exists := m.byQRCode[p.QRCode]; exists {
		return nil, domain.ErrDuplicateQRCode
	}

	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	p.CreatedAt = time.Now()
	p.UpdatedAt = time.Now()
	p.IsActive = true

	m.byID[p.ID] = p
	m.bySKU[p.SKU] = p
	m.byQRCode[p.QRCode] = p

	return p, nil
}

func (m *mockProductRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
	p, exists := m.byID[id]
	if !exists || !p.IsActive {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (m *mockProductRepo) GetByQRCode(ctx context.Context, qrCode string) (*domain.Product, error) {
	p, exists := m.byQRCode[qrCode]
	if !exists || !p.IsActive {
		return nil, domain.ErrProductNotFound
	}
	return p, nil
}

func (m *mockProductRepo) List(ctx context.Context, filter ProductFilter) ([]*domain.Product, int, error) {
	var result []*domain.Product
	for _, p := range m.byID {
		if !p.IsActive {
			continue
		}
		if filter.Search != "" {
			term := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(p.Name), term) && !strings.Contains(strings.ToLower(p.SKU), term) {
				continue
			}
		}
		if filter.LowStock && !p.IsLowStock() {
			continue
		}
		result = append(result, p)
	}
	return result, len(result), nil
}

func (m *mockProductRepo) Update(ctx context.Context, p *domain.Product) (*domain.Product, error) {
	existing, exists := m.byID[p.ID]
	if !exists || !existing.IsActive {
		return nil, domain.ErrProductNotFound
	}
	p.UpdatedAt = time.Now()
	m.byID[p.ID] = p
	return p, nil
}

func (m *mockProductRepo) Deactivate(ctx context.Context, id uuid.UUID) error {
	existing, exists := m.byID[id]
	if !exists || !existing.IsActive {
		return domain.ErrProductNotFound
	}
	existing.IsActive = false
	return nil
}

// 1. Tests for Seller Cost Price Masking vs Owner Visibility
func TestProductUsecase_CostPriceMasking(t *testing.T) {
	ctx := context.Background()
	repo := newMockProductRepo()
	usecase := NewProductUsecase(repo)

	// Seed product
	created, err := usecase.CreateProduct(ctx, CreateProductDTO{
		Name:          "AirPods Pro Max",
		SKU:           "AP-MAX-001",
		QRCode:        "QR-AP-MAX-001",
		CostPrice:     decimal.NewFromFloat(350.00),
		SellingPrice:  decimal.NewFromFloat(549.99),
		StockQuantity: 15,
		MinStockAlert: 5,
	})
	if err != nil {
		t.Fatalf("unexpected error creating product: %v", err)
	}

	prodID, err := uuid.Parse(created.ID)
	if err != nil {
		t.Fatalf("failed to parse UUID: %v", err)
	}

	// Case A: Owner retrieves product by ID -> sees real cost_price
	ownerResp, err := usecase.GetByID(ctx, prodID, "owner")
	if err != nil {
		t.Fatalf("failed to get product as owner: %v", err)
	}
	if ownerResp.CostPrice != "350.00" {
		t.Errorf("expected owner to see real cost_price '350.00', got %q", ownerResp.CostPrice)
	}

	// Case B: Seller retrieves product by ID -> sees masked cost_price "0.00"
	sellerResp, err := usecase.GetByID(ctx, prodID, "seller")
	if err != nil {
		t.Fatalf("failed to get product as seller: %v", err)
	}
	if sellerResp.CostPrice != "0.00" {
		t.Errorf("CRITICAL SECURITY: expected seller to see masked cost_price '0.00', got %q", sellerResp.CostPrice)
	}

	// Case C: Seller retrieves product by QR Code -> sees masked cost_price "0.00"
	sellerQRResp, err := usecase.GetByQRCode(ctx, "QR-AP-MAX-001", "seller")
	if err != nil {
		t.Fatalf("failed to get product by QR as seller: %v", err)
	}
	if sellerQRResp.CostPrice != "0.00" {
		t.Errorf("CRITICAL SECURITY: expected seller to see masked cost_price '0.00' via QR lookup, got %q", sellerQRResp.CostPrice)
	}

	// Case D: Seller lists products -> sees masked cost_price "0.00"
	sellerList, total, err := usecase.ListProducts(ctx, ProductFilter{}, "seller")
	if err != nil {
		t.Fatalf("failed to list products as seller: %v", err)
	}
	if total != 1 || len(sellerList) != 1 {
		t.Fatalf("expected 1 product in list, got %d", total)
	}
	if sellerList[0].CostPrice != "0.00" {
		t.Errorf("CRITICAL SECURITY: expected seller to see masked cost_price '0.00' in list, got %q", sellerList[0].CostPrice)
	}

	// Case E: Owner lists products -> sees real cost_price "350.00"
	ownerList, _, err := usecase.ListProducts(ctx, ProductFilter{}, "owner")
	if err != nil {
		t.Fatalf("failed to list products as owner: %v", err)
	}
	if ownerList[0].CostPrice != "350.00" {
		t.Errorf("expected owner to see real cost_price '350.00' in list, got %q", ownerList[0].CostPrice)
	}
}

// 2. Tests for Invariants & Validations
func TestProductUsecase_ValidationInvariants(t *testing.T) {
	ctx := context.Background()
	repo := newMockProductRepo()
	usecase := NewProductUsecase(repo)

	// Negative cost price
	_, err := usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "Invalid Cost",
		SKU:          "INV-01",
		CostPrice:    decimal.NewFromFloat(-5.00),
		SellingPrice: decimal.NewFromFloat(10.00),
	})
	if err != domain.ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice, got %v", err)
	}

	// Negative selling price
	_, err = usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "Invalid Selling",
		SKU:          "INV-02",
		CostPrice:    decimal.NewFromFloat(5.00),
		SellingPrice: decimal.NewFromFloat(-10.00),
	})
	if err != domain.ErrInvalidPrice {
		t.Errorf("expected ErrInvalidPrice, got %v", err)
	}

	// Negative stock quantity
	_, err = usecase.CreateProduct(ctx, CreateProductDTO{
		Name:          "Invalid Stock",
		SKU:           "INV-03",
		CostPrice:     decimal.NewFromFloat(5.00),
		SellingPrice:  decimal.NewFromFloat(10.00),
		StockQuantity: -1,
	})
	if err != domain.ErrInvalidQuantity {
		t.Errorf("expected ErrInvalidQuantity, got %v", err)
	}

	// Empty Name
	_, err = usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "",
		SKU:          "INV-04",
		CostPrice:    decimal.NewFromFloat(5.00),
		SellingPrice: decimal.NewFromFloat(10.00),
	})
	if err != domain.ErrValidation {
		t.Errorf("expected ErrValidation, got %v", err)
	}
}

// 3. Tests for QR Code Image Generation
func TestProductUsecase_GenerateQRCodePNG(t *testing.T) {
	ctx := context.Background()
	repo := newMockProductRepo()
	usecase := NewProductUsecase(repo)

	created, err := usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "QR Test Item",
		SKU:          "QR-TEST-001",
		QRCode:       "BARCODE-123456789",
		CostPrice:    decimal.NewFromFloat(10.00),
		SellingPrice: decimal.NewFromFloat(15.00),
	})
	if err != nil {
		t.Fatalf("unexpected error creating product: %v", err)
	}

	prodID, _ := uuid.Parse(created.ID)
	pngBytes, err := usecase.GenerateQRCodePNG(ctx, prodID)
	if err != nil {
		t.Fatalf("failed to generate QR code PNG: %v", err)
	}

	if len(pngBytes) == 0 {
		t.Fatal("expected non-empty PNG bytes")
	}

	// Validate PNG magic number signature: \x89PNG\r\n\x1a\n
	pngHeader := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if !bytes.HasPrefix(pngBytes, pngHeader) {
		t.Errorf("generated bytes do not have a valid PNG header")
	}
}

// 4. Tests for Duplicate SKU and QR Code collisions
func TestProductUsecase_DuplicateSKUAndQRCode(t *testing.T) {
	ctx := context.Background()
	repo := newMockProductRepo()
	usecase := NewProductUsecase(repo)

	_, err := usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "First Item",
		SKU:          "DUP-SKU-001",
		QRCode:       "DUP-QR-001",
		CostPrice:    decimal.NewFromFloat(10.00),
		SellingPrice: decimal.NewFromFloat(20.00),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Attempt same SKU
	_, err = usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "Second Item",
		SKU:          "DUP-SKU-001",
		QRCode:       "DIFFERENT-QR",
		CostPrice:    decimal.NewFromFloat(10.00),
		SellingPrice: decimal.NewFromFloat(20.00),
	})
	if err != domain.ErrDuplicateSKU {
		t.Errorf("expected ErrDuplicateSKU, got %v", err)
	}

	// Attempt same QRCode
	_, err = usecase.CreateProduct(ctx, CreateProductDTO{
		Name:         "Third Item",
		SKU:          "DIFFERENT-SKU",
		QRCode:       "DUP-QR-001",
		CostPrice:    decimal.NewFromFloat(10.00),
		SellingPrice: decimal.NewFromFloat(20.00),
	})
	if err != domain.ErrDuplicateQRCode {
		t.Errorf("expected ErrDuplicateQRCode, got %v", err)
	}
}
