package usecase

import (
	"context"
	"testing"
	"time"

	authDomain "github.com/UmedjonQurbonov/CRM/internal/modules/auth/domain"
	"github.com/UmedjonQurbonov/CRM/internal/modules/order/domain"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockUserRepoForOrder struct {
	users map[uuid.UUID]*authDomain.User
}

func (m *mockUserRepoForOrder) Create(ctx context.Context, u *authDomain.User) error {
	m.users[u.ID] = u
	return nil
}
func (m *mockUserRepoForOrder) GetByID(ctx context.Context, id uuid.UUID) (*authDomain.User, error) {
	u, ok := m.users[id]
	if !ok {
		return nil, authDomain.ErrUserNotFound
	}
	return u, nil
}
func (m *mockUserRepoForOrder) GetByPhone(ctx context.Context, phone string) (*authDomain.User, error) {
	return nil, nil
}
func (m *mockUserRepoForOrder) ListSellers(ctx context.Context) ([]*authDomain.User, error) {
	return nil, nil
}
func (m *mockUserRepoForOrder) UpdateCommissionRate(ctx context.Context, id uuid.UUID, rate decimal.Decimal) error {
	return nil
}
func (m *mockUserRepoForOrder) Count(ctx context.Context) (int64, error) {
	return int64(len(m.users)), nil
}

type mockProductStock struct {
	Stock     int
	CostPrice decimal.Decimal
	SalePrice decimal.Decimal
	Name      string
}

type mockOrderRepo struct {
	orders   map[uuid.UUID]*domain.Order
	products map[uuid.UUID]*mockProductStock
}

func newMockOrderRepo() *mockOrderRepo {
	return &mockOrderRepo{
		orders:   make(map[uuid.UUID]*domain.Order),
		products: make(map[uuid.UUID]*mockProductStock),
	}
}

func (m *mockOrderRepo) CreateOrderTx(
	ctx context.Context,
	sellerID uuid.UUID,
	paymentMethod string,
	commissionRate decimal.Decimal,
	items []CartItemInput,
) (*domain.Order, error) {
	if len(items) == 0 {
		return nil, domain.ErrEmptyCart
	}

	totalAmount := decimal.Zero
	totalCost := decimal.Zero
	var orderItems []domain.OrderItem
	orderID := uuid.New()

	for _, item := range items {
		prod, ok := m.products[item.ProductID]
		if !ok || prod.Stock < item.Quantity {
			return nil, domain.ErrInsufficientStock
		}
		prod.Stock -= item.Quantity

		qtyDec := decimal.NewFromInt(int64(item.Quantity))
		subtotal := prod.SalePrice.Mul(qtyDec)
		totalAmount = totalAmount.Add(subtotal)
		totalCost = totalCost.Add(prod.CostPrice.Mul(qtyDec))

		orderItems = append(orderItems, domain.OrderItem{
			ID:            uuid.New(),
			OrderID:       orderID,
			ProductID:     item.ProductID,
			ProductName:   prod.Name,
			Quantity:      item.Quantity,
			UnitPrice:     prod.SalePrice,
			UnitCostPrice: prod.CostPrice,
			Subtotal:      subtotal,
		})
	}

	commEarned := totalAmount.Mul(commissionRate).Div(decimal.NewFromInt(100)).Round(2)

	order := &domain.Order{
		ID:                     orderID,
		SellerID:               sellerID,
		TotalAmount:            totalAmount,
		TotalCost:              totalCost,
		CommissionRateSnapshot: commissionRate,
		CommissionEarned:       commEarned,
		PaymentMethod:          paymentMethod,
		Status:                 domain.StatusCompleted,
		Items:                  orderItems,
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}

	m.orders[orderID] = order
	return order, nil
}

func (m *mockOrderRepo) RefundOrderTx(ctx context.Context, orderID uuid.UUID) (*domain.Order, error) {
	order, ok := m.orders[orderID]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	if order.Status == domain.StatusRefunded {
		return nil, domain.ErrOrderAlreadyRefunded
	}

	for _, item := range order.Items {
		if prod, exists := m.products[item.ProductID]; exists {
			prod.Stock += item.Quantity
		}
	}

	order.Status = domain.StatusRefunded
	order.UpdatedAt = time.Now()
	return order, nil
}

func (m *mockOrderRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	o, ok := m.orders[id]
	if !ok {
		return nil, domain.ErrOrderNotFound
	}
	return o, nil
}

func (m *mockOrderRepo) List(ctx context.Context, filter OrderFilter) ([]*domain.Order, int, error) {
	var result []*domain.Order
	for _, o := range m.orders {
		if filter.SellerID != nil && o.SellerID != *filter.SellerID {
			continue
		}
		result = append(result, o)
	}
	return result, len(result), nil
}

func TestOrderUsecase_Checkout_OwnerZeroCommission(t *testing.T) {
	ctx := context.Background()
	userRepo := &mockUserRepoForOrder{users: make(map[uuid.UUID]*authDomain.User)}
	orderRepo := newMockOrderRepo()

	ownerID := uuid.New()
	prodID := uuid.New()

	orderRepo.products[prodID] = &mockProductStock{
		Stock:     10,
		CostPrice: decimal.NewFromFloat(50.00),
		SalePrice: decimal.NewFromFloat(100.00),
		Name:      "AirPods Pro",
	}

	usecase := NewOrderUsecase(orderRepo, userRepo)

	req := CheckoutRequestDTO{
		PaymentMethod: "cash",
		Items: []CartItemDTO{
			{ProductID: prodID.String(), Quantity: 2},
		},
	}

	order, err := usecase.Checkout(ctx, ownerID, "owner", req)
	if err != nil {
		t.Fatalf("checkout failed: %v", err)
	}

	if order.TotalAmount != "200.00" {
		t.Errorf("expected total_amount 200.00, got %s", order.TotalAmount)
	}
	if order.CommissionRateSnapshot != "0.00" {
		t.Errorf("expected owner commission rate snapshot 0.00, got %s", order.CommissionRateSnapshot)
	}
	if order.CommissionEarned != "0.00" {
		t.Errorf("expected owner commission earned 0.00, got %s", order.CommissionEarned)
	}
	if orderRepo.products[prodID].Stock != 8 {
		t.Errorf("expected remaining stock 8, got %d", orderRepo.products[prodID].Stock)
	}
}

func TestOrderUsecase_Checkout_SellerCommissionEarned(t *testing.T) {
	ctx := context.Background()
	userRepo := &mockUserRepoForOrder{users: make(map[uuid.UUID]*authDomain.User)}
	orderRepo := newMockOrderRepo()

	sellerID := uuid.New()
	userRepo.users[sellerID] = &authDomain.User{
		ID:             sellerID,
		Name:           "Seller",
		Role:           authDomain.RoleSeller,
		CommissionRate: decimal.NewFromFloat(5.00), // 5%
	}

	prodID := uuid.New()
	orderRepo.products[prodID] = &mockProductStock{
		Stock:     5,
		CostPrice: decimal.NewFromFloat(100.00),
		SalePrice: decimal.NewFromFloat(200.00),
		Name:      "Headphones",
	}

	usecase := NewOrderUsecase(orderRepo, userRepo)

	req := CheckoutRequestDTO{
		PaymentMethod: "card",
		Items: []CartItemDTO{
			{ProductID: prodID.String(), Quantity: 2}, // 2 * 200 = 400.00
		},
	}

	order, err := usecase.Checkout(ctx, sellerID, "seller", req)
	if err != nil {
		t.Fatalf("seller checkout failed: %v", err)
	}

	if order.TotalAmount != "400.00" {
		t.Errorf("expected total_amount 400.00, got %s", order.TotalAmount)
	}
	if order.CommissionRateSnapshot != "5.00" {
		t.Errorf("expected commission rate snapshot 5.00, got %s", order.CommissionRateSnapshot)
	}
	// 5% of 400.00 is 20.00
	if order.CommissionEarned != "20.00" {
		t.Errorf("expected commission earned 20.00, got %s", order.CommissionEarned)
	}
}

func TestOrderUsecase_Checkout_InsufficientStock(t *testing.T) {
	ctx := context.Background()
	userRepo := &mockUserRepoForOrder{users: make(map[uuid.UUID]*authDomain.User)}
	orderRepo := newMockOrderRepo()

	sellerID := uuid.New()
	userRepo.users[sellerID] = &authDomain.User{
		ID:             sellerID,
		CommissionRate: decimal.NewFromFloat(5.00),
	}

	prodID := uuid.New()
	orderRepo.products[prodID] = &mockProductStock{
		Stock:     2,
		CostPrice: decimal.NewFromFloat(10.00),
		SalePrice: decimal.NewFromFloat(20.00),
	}

	usecase := NewOrderUsecase(orderRepo, userRepo)

	req := CheckoutRequestDTO{
		PaymentMethod: "cash",
		Items: []CartItemDTO{
			{ProductID: prodID.String(), Quantity: 5}, // requested 5, stock 2
		},
	}

	_, err := usecase.Checkout(ctx, sellerID, "seller", req)
	if err != domain.ErrInsufficientStock {
		t.Errorf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestOrderUsecase_Refund_OwnerOnlyAndStockRestoration(t *testing.T) {
	ctx := context.Background()
	userRepo := &mockUserRepoForOrder{users: make(map[uuid.UUID]*authDomain.User)}
	orderRepo := newMockOrderRepo()

	sellerID := uuid.New()
	prodID := uuid.New()
	orderRepo.products[prodID] = &mockProductStock{
		Stock:     10,
		CostPrice: decimal.NewFromFloat(10.00),
		SalePrice: decimal.NewFromFloat(20.00),
	}

	usecase := NewOrderUsecase(orderRepo, userRepo)

	// Buy 3 items
	order, _ := usecase.Checkout(ctx, sellerID, "owner", CheckoutRequestDTO{
		PaymentMethod: "cash",
		Items:         []CartItemDTO{{ProductID: prodID.String(), Quantity: 3}},
	})
	if orderRepo.products[prodID].Stock != 7 {
		t.Fatalf("expected stock 7 after purchase, got %d", orderRepo.products[prodID].Stock)
	}

	orderUUID, _ := uuid.Parse(order.ID)

	// Case A: Seller tries to refund -> rejected
	_, err := usecase.RefundOrder(ctx, orderUUID, "seller")
	if err != domain.ErrForbidden {
		t.Errorf("expected ErrForbidden for seller refund, got %v", err)
	}

	// Case B: Owner refunds -> stock restored
	refunded, err := usecase.RefundOrder(ctx, orderUUID, "owner")
	if err != nil {
		t.Fatalf("expected owner refund success, got: %v", err)
	}
	if refunded.Status != domain.StatusRefunded {
		t.Errorf("expected status refunded, got %s", refunded.Status)
	}
	if orderRepo.products[prodID].Stock != 10 {
		t.Errorf("expected stock restored to 10, got %d", orderRepo.products[prodID].Stock)
	}

	// Case C: Duplicate refund -> rejected
	_, err = usecase.RefundOrder(ctx, orderUUID, "owner")
	if err != domain.ErrOrderAlreadyRefunded {
		t.Errorf("expected ErrOrderAlreadyRefunded, got %v", err)
	}
}

func TestOrderUsecase_SellerDataIsolation(t *testing.T) {
	ctx := context.Background()
	userRepo := &mockUserRepoForOrder{users: make(map[uuid.UUID]*authDomain.User)}
	orderRepo := newMockOrderRepo()

	sellerA := uuid.New()
	sellerB := uuid.New()
	prodID := uuid.New()
	orderRepo.products[prodID] = &mockProductStock{
		Stock:     10,
		SalePrice: decimal.NewFromFloat(20.00),
	}

	usecase := NewOrderUsecase(orderRepo, userRepo)

	// Seller A creates order
	orderA, _ := usecase.Checkout(ctx, sellerA, "owner", CheckoutRequestDTO{
		PaymentMethod: "cash",
		Items:         []CartItemDTO{{ProductID: prodID.String(), Quantity: 1}},
	})
	orderUUID, _ := uuid.Parse(orderA.ID)

	// Seller B attempts to view Seller A's order -> forbidden
	_, err := usecase.GetOrderByID(ctx, orderUUID, sellerB, "seller")
	if err != domain.ErrForbidden {
		t.Errorf("expected ErrForbidden for seller viewing other seller's order, got %v", err)
	}

	// Seller A viewing own order -> success
	viewA, err := usecase.GetOrderByID(ctx, orderUUID, sellerA, "seller")
	if err != nil || viewA.ID != orderA.ID {
		t.Errorf("expected seller A to view own order, got error: %v", err)
	}

	// Owner viewing Seller A's order -> success
	viewOwner, err := usecase.GetOrderByID(ctx, orderUUID, uuid.New(), "owner")
	if err != nil || viewOwner.ID != orderA.ID {
		t.Errorf("expected owner to view order, got error: %v", err)
	}
}
