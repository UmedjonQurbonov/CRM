package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/UmedjonQurbonov/CRM/internal/modules/product/domain"
	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

// ProductUsecase coordinates product catalog and inventory management workflows.
type ProductUsecase struct {
	repo ProductRepository
}

func NewProductUsecase(repo ProductRepository) *ProductUsecase {
	return &ProductUsecase{repo: repo}
}

// CreateProduct creates a new catalog item.
func (u *ProductUsecase) CreateProduct(ctx context.Context, dto CreateProductDTO) (*ProductResponseDTO, error) {
	name := strings.TrimSpace(dto.Name)
	sku := strings.TrimSpace(dto.SKU)
	qrCode := strings.TrimSpace(dto.QRCode)
	if qrCode == "" {
		qrCode = sku
	}

	minStock := dto.MinStockAlert
	if minStock <= 0 {
		minStock = 5
	}

	product := &domain.Product{
		ID:            uuid.New(),
		Name:          name,
		SKU:           sku,
		QRCode:        qrCode,
		CostPrice:     dto.CostPrice,
		SellingPrice:  dto.SellingPrice,
		StockQuantity: dto.StockQuantity,
		MinStockAlert: minStock,
	}

	if err := product.Validate(); err != nil {
		return nil, err
	}

	created, err := u.repo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(created, "owner"), nil
}

// UpdateProduct updates attributes of an existing catalog item.
func (u *ProductUsecase) UpdateProduct(ctx context.Context, dto UpdateProductDTO) (*ProductResponseDTO, error) {
	if dto.ID == uuid.Nil {
		return nil, domain.ErrValidation
	}

	existing, err := u.repo.GetByID(ctx, dto.ID)
	if err != nil {
		return nil, err
	}

	existing.Name = strings.TrimSpace(dto.Name)
	existing.SKU = strings.TrimSpace(dto.SKU)
	if strings.TrimSpace(dto.QRCode) != "" {
		existing.QRCode = strings.TrimSpace(dto.QRCode)
	}
	existing.CostPrice = dto.CostPrice
	existing.SellingPrice = dto.SellingPrice
	existing.StockQuantity = dto.StockQuantity
	existing.MinStockAlert = dto.MinStockAlert

	if err := existing.Validate(); err != nil {
		return nil, err
	}

	updated, err := u.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(updated, "owner"), nil
}

// GetByID retrieves a product by its unique identifier, applying role masking.
func (u *ProductUsecase) GetByID(ctx context.Context, id uuid.UUID, callerRole string) (*ProductResponseDTO, error) {
	if id == uuid.Nil {
		return nil, domain.ErrValidation
	}

	product, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(product, callerRole), nil
}

// GetByQRCode retrieves a product matching the scanned QR code, applying role masking.
func (u *ProductUsecase) GetByQRCode(ctx context.Context, qrCode string, callerRole string) (*ProductResponseDTO, error) {
	qrCode = strings.TrimSpace(qrCode)
	if qrCode == "" {
		return nil, domain.ErrValidation
	}

	product, err := u.repo.GetByQRCode(ctx, qrCode)
	if err != nil {
		return nil, err
	}

	return ToProductResponseDTO(product, callerRole), nil
}

// ListProducts returns paginated catalog items with role-based masking.
func (u *ProductUsecase) ListProducts(ctx context.Context, filter ProductFilter, callerRole string) ([]*ProductResponseDTO, int, error) {
	products, total, err := u.repo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	return ToProductResponseDTOList(products, callerRole), total, nil
}

// GenerateQRCodePNG generates a PNG barcode representation of the product's QR code string.
func (u *ProductUsecase) GenerateQRCodePNG(ctx context.Context, id uuid.UUID) ([]byte, error) {
	if id == uuid.Nil {
		return nil, domain.ErrValidation
	}

	product, err := u.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	pngBytes, err := qrcode.Encode(product.QRCode, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code PNG: %w", err)
	}

	return pngBytes, nil
}
