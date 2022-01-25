package repository

import (
	"context"
	"fmt"
	"github.com/EgorBessonov/gRPC/internal/model"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

// PostgresRepository type replies for accessing to postgres database
type PostgresRepository struct {
	DBconn *pgxpool.Pool
}

// Save save Order object into postgresql database
func (rps PostgresRepository) Save(ctx context.Context, order *model.Order) error {
	log.WithFields(log.Fields{
		"orderID":   order.OrderID,
		"orderName": order.OrderName,
	}).Debugf("repository: create order")
	_, err := rps.DBconn.Exec(ctx, `insert into orders (orderID, orderName, orderCost, isDelivered) 
		values ($1, $2, $3, $4)`, order.OrderID, order.OrderName, order.OrderCost, order.IsDelivered)
	if err != nil {
		return fmt.Errorf("postgres repository: can't save order", err)
	}
	return nil
}

// Get method returns Order object from postgresql database
// with selection by OrderID
func (rps PostgresRepository) Get(ctx context.Context, orderID string) (*model.Order, error) {
	log.WithFields(log.Fields{
		"orderID": orderID,
	}).Debugf("repository: get order")
	var order model.Order
	err := rps.DBconn.QueryRow(ctx, `select orderID, orderName, orderCost, isDelivered from orders 
		where orderID=$1`, orderID).Scan(&order.OrderID, &order.OrderName, &order.OrderCost, &order.IsDelivered)
	if err != nil {
		return nil, fmt.Errorf("postgres repository: can't get order - %w", err)
	}
	return &order, nil
}

// Update method update Order object from postgresql database
// with selection by OrderID
func (rps PostgresRepository) Update(ctx context.Context, order *model.Order) error {
	log.WithFields(log.Fields{
		"orderID":   order.OrderID,
		"orderName": order.OrderName,
	}).Debugf("postgres repository: update order")
	_, err := rps.DBconn.Exec(ctx, `update orders
		set orderName=$2, orderCost=$3, isDelivered=$4
		where orderID=$1`, order.OrderID, order.OrderName, order.OrderCost, order.IsDelivered)
	if err != nil {
		return fmt.Errorf("postgres repository: can't update order - %w", err)
	}
	return nil
}

// Delete method delete Order object from postgresql database
// with selection by OrderID
func (rps PostgresRepository) Delete(ctx context.Context, orderID string) error {
	log.WithFields(log.Fields{
		"orderID": orderID,
	}).Debugf("postgres repository: delete order")
	_, err := rps.DBconn.Exec(ctx, "delete from orders where orderID=$1", orderID)
	if err != nil {
		return fmt.Errorf("repository: can't delete order - %w", err)
	}
	return nil
}

// SaveAuthUser method saves authentication info about user into
// postgres database
func (rps PostgresRepository) SaveAuthUser(ctx context.Context, authUser *model.AuthUser) error {
	log.WithFields(log.Fields{
		"userID":   authUser.UserUUID,
		"userName": authUser.UserName,
	}).Debugf("postgres repository: save authUser")
	_, err := rps.DBconn.Exec(ctx, `insert into authusers (username, email, password) 
		values($1, $2, $3)`, authUser.UserName, authUser.Email, authUser.Password)
	if err != nil {
		return fmt.Errorf("postgres repository: can't save authUser - %w", err)
	}
	return nil
}

// GetAuthUser method returns authentication info about user from
// postgres database with selection by email
func (rps PostgresRepository) GetAuthUser(ctx context.Context, email string) (*model.AuthUser, error) {
	log.WithFields(log.Fields{
		"email": email,
	}).Debugf("postgres repository: get authUser by email")
	var authUser model.AuthUser
	err := rps.DBconn.QueryRow(ctx, `select useruuid, username, email, password from authusers
		where email=$1`, email).Scan(&authUser.UserUUID, &authUser.UserName, &authUser.Email, &authUser.Password)
	if err != nil {
		return nil, fmt.Errorf("repository: can't get authUser - %w", err)
	}
	return &authUser, nil
}

// GetAuthUserByID method returns authentication info about user from
// postgres database with selection by id
func (rps PostgresRepository) GetAuthUserByID(ctx context.Context, userUUID string) (*model.AuthUser, error) {
	log.WithFields(log.Fields{
		"userID": userUUID,
	}).Debugf("postgres repository: get authUser by id")
	var authUser model.AuthUser
	err := rps.DBconn.QueryRow(ctx, `select useruuid, username, email, password, refreshtoken from authusers
		where useruuid=$1`, userUUID).Scan(&authUser.UserUUID, &authUser.UserName, &authUser.Email, &authUser.Password, &authUser.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("repository: can't get authUser by ID - %w", err)
	}
	return &authUser, nil
}

// UpdateAuthUser is method to set refresh token into authuser info
func (rps PostgresRepository) UpdateAuthUser(ctx context.Context, email, refreshToken string) error {
	log.WithFields(log.Fields{
		"email":        email,
		"refreshToken": refreshToken,
	}).Debugf("postgres repository: update authUser")
	_, err := rps.DBconn.Exec(ctx, `update authusers
		set refreshtoken=$2
		where email=$1`, email, refreshToken)
	if err != nil {
		return fmt.Errorf("repository: can't update authUser - %w", err)
	}
	return nil
}

// CloseDBConnection is using to close current postgres database connection
func (rps PostgresRepository) CloseDBConnection() error {
	rps.DBconn.Close()
	return fmt.Errorf("repository: database connection closed")
}
