package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/skip2/go-qrcode"
)

// ProductUsecase encapsulates business operations for inventory catalog.
type ProductUsecase struct {
	productRepo ProductRepository
}

// NewProductUsecase creates a new ProductUsecase instance.
func NewProductUsecase(productRepo ProductRepository) *ProductUsecase {
	return &ProductUsecase{
		productRepo: productRepo,
	}
}

// CreateProduct adds a new item to catalog.
func (u *ProductUsecase) CreateProduct(ctx context.Context, dto CreateProductDTO) (*ProductResponseDTO, error) {
	sku := strings.TrimSpace(dto.SKU)
	qrCode := strings.TrimSpace(dto.QRCode)
	if qrCode == "" {
		qrCode = sku
	}

	product := &domain.Product{
		Name:          strings.TrimSpace(dto.Name),
		SKU:           sku,
		QRCode:        qrCode,
		Category:      strings.TrimSpace(dto.Category),
		ImageURL:      strings.TrimSpace(dto.ImageURL),
		CostPrice:     dto.CostPrice,
		SellingPrice:  dto.SellingPrice,
		StockQuantity: dto.StockQuantity,
		MinStockAlert: dto.MinStockAlert,
	}

	if err := product.Validate(); err != nil {
		return nil, err
	}

	created, err := u.productRepo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(created, "owner"), nil
}

// GetByID retrieves a product by its UUID.
func (u *ProductUsecase) GetByID(ctx context.Context, id uuid.UUID, callerRole string) (*ProductResponseDTO, error) {
	if id == uuid.Nil {
		return nil, domain.ErrValidation
	}

	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(product, callerRole), nil
}

// GetByQRCode retrieves a product by its barcode.
func (u *ProductUsecase) GetByQRCode(ctx context.Context, qrCode string, callerRole string) (*ProductResponseDTO, error) {
	cleanQR := strings.TrimSpace(qrCode)
	if cleanQR == "" {
		return nil, domain.ErrValidation
	}

	product, err := u.productRepo.GetByQRCode(ctx, cleanQR)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(product, callerRole), nil
}

// ListProducts retrieves catalog items matching query parameters.
func (u *ProductUsecase) ListProducts(
	ctx context.Context,
	filter ProductFilter,
	callerRole string,
) ([]*ProductResponseDTO, int, error) {
	products, total, err := u.productRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return ToProductResponseDTOList(products, callerRole), total, nil
}

// UpdateProduct updates catalog properties of a product.
func (u *ProductUsecase) UpdateProduct(ctx context.Context, dto UpdateProductDTO) (*ProductResponseDTO, error) {
	if dto.ID == uuid.Nil {
		return nil, domain.ErrValidation
	}

	product, err := u.productRepo.GetByID(ctx, dto.ID)
	if err != nil {
		return nil, err
	}

	product.Name = strings.TrimSpace(dto.Name)
	product.SKU = strings.TrimSpace(dto.SKU)
	product.QRCode = strings.TrimSpace(dto.QRCode)
	product.Category = strings.TrimSpace(dto.Category)
	if dto.ImageURL != "" {
		product.ImageURL = strings.TrimSpace(dto.ImageURL)
	}
	product.CostPrice = dto.CostPrice
	product.SellingPrice = dto.SellingPrice
	product.StockQuantity = dto.StockQuantity
	product.MinStockAlert = dto.MinStockAlert

	if err := product.Validate(); err != nil {
		return nil, err
	}

	updated, err := u.productRepo.Update(ctx, product)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(updated, "owner"), nil
}

// RestockProduct increases stock quantity and optionally updates cost price.
func (u *ProductUsecase) RestockProduct(ctx context.Context, dto RestockProductDTO) (*ProductResponseDTO, error) {
	if dto.ID == uuid.Nil || dto.Quantity <= 0 {
		return nil, domain.ErrInvalidQuantity
	}
	if dto.CostPrice != nil && dto.CostPrice.IsNegative() {
		return nil, domain.ErrInvalidPrice
	}

	product, err := u.productRepo.Restock(ctx, dto.ID, dto.Quantity, dto.CostPrice)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(product, "owner"), nil
}

// UpdateProductImage sets the image URL for a product.
func (u *ProductUsecase) UpdateProductImage(ctx context.Context, id uuid.UUID, imageURL string) error {
	if id == uuid.Nil {
		return domain.ErrValidation
	}
	return u.productRepo.UpdateImage(ctx, id, strings.TrimSpace(imageURL))
}

// GetTotalStockProfit calculates total potential profit across all inventory (Owner only).
func (u *ProductUsecase) GetTotalStockProfit(ctx context.Context, callerRole string) (decimal.Decimal, error) {
	if callerRole != "owner" {
		return decimal.Zero, nil
	}
	return u.productRepo.GetTotalStockProfit(ctx)
}

// GenerateQRCodePNG dynamically generates and returns PNG bytes for a product QR code.
func (u *ProductUsecase) GenerateQRCodePNG(ctx context.Context, id uuid.UUID) ([]byte, error) {
	product, err := u.productRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	png, err := qrcode.Encode(product.QRCode, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to encode QR code: %w", err)
	}

	return png, nil
}
