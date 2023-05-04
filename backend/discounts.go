// Package src
/*
Copyright © 2023 JOSEF MUELLER
*/
package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
)

// Discount is a struct for discounts
type Discount struct {
	ID               primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"` // id of the product
	Title            string             `json:"title" bson:"title"`                 // title of the product
	Price            string             `json:"price" bson:"price"`                 // price of the product
	ImageUrl         string             `json:"imageUrl" bson:"imageUrl"`
	ValidUntil       int                `json:"validUntil" bson:"validUntil"` // unix timestamp
	MarketName       string             `json:"marketName" bson:"marketName"`
	InternalMarketID string             `json:"internalMarketId,omitempty" bson:"internalMarketId,omitempty"`
	Tags             []string           `json:"-" bson:"_tags"`
}

// HandleGetDiscountsByCity gets called by router
// Calls getDiscountsFromDBOrAPI and handles the context
func (server *TasteBuddyServer) HandleGetDiscountsByCity(context *gin.Context) {
	city := context.Param("city")
	server.Log("HandleGetDiscountsByCity", "Get discounts for city "+city)
	if discounts, err := server.GetDiscountsByCity(city); err != nil {
		server.LogError("HandleGetDiscountsByCity + "+city, err)
		ServerError(context, true)
	} else {
		Success(context, discounts)
	}
}

// HandleGetAllDiscounts gets called by router
// Calls getDiscountsFromDB
func (server *TasteBuddyServer) HandleGetAllDiscounts(context *gin.Context) {
	server.Log("HandleGetAllDiscounts", "Get all discounts")
	if discounts, err := server.GetAllDiscounts(); err != nil {
		server.LogError("HandleGetAllDiscounts", err)
		ServerError(context, true)
	} else {
		Success(context, discounts)
	}
}

// GetDiscountsCollection gets discounts collection from database
func (app *TasteBuddyApp) GetDiscountsCollection() *mongo.Collection {
	return app.client.Database("tastebuddy").Collection("discounts")
}

// GetDiscountsByCity gets discounts by city from database
func (app *TasteBuddyApp) GetDiscountsByCity(city string) ([]Discount, error) {
	ctx := DefaultContext()

	// get marketIds for city
	marketIds := app.GetAllMarketIDsByCity(city)
	if len(marketIds) == 0 {
		return []Discount{}, errors.New("No markets found for " + city)
	}

	// get discounts for markets
	discountsFromDB, err := app.GetDiscountsCollection().Find(ctx, bson.D{{Key: "internalMarketId", Value: bson.D{{Key: "$in", Value: marketIds}}}})
	if err != nil {
		app.LogError("GetDiscountsByCity + "+city, err)
		return []Discount{}, err
	}

	// get discounts from database
	var discounts []Discount
	if err := discountsFromDB.All(ctx, &discounts); err != nil {
		app.LogError("GetDiscountsByCity + "+city, err)
		return []Discount{}, err
	}

	// there can be multiple copies of the same discount after the filter operation above
	var discountsMap = make(map[string]Discount)
	var filteredDiscounts []Discount
	for _, discount := range discounts {
		// only add discount if it is not already in the map
		if _, ok := discountsMap[discount.Title]; !ok {
			discountsMap[discount.Title] = discount
		}
	}
	// convert map to array
	for _, discount := range discountsMap {
		filteredDiscounts = append(filteredDiscounts, discount)
	}
	discounts = filteredDiscounts

	app.Log("GetDiscountsByCity + "+city, "Return "+strconv.Itoa(len(discounts))+" discounts from database")
	return discounts, nil
}

// GetDiscountsFromAPI gets all discounts for a market from the market's API
func (app *TasteBuddyApp) GetDiscountsFromAPI(market *Market) ([]Discount, error) {
	switch market.Distributor {
	case "edeka":
		return app.GetEdekaDiscounts(*market)
	case "rewe":
		return app.GetReweDiscounts(*market)
	}
	return []Discount{}, nil
}

// GetAllDiscounts gets all discounts from database
func (app *TasteBuddyApp) GetAllDiscounts() ([]Discount, error) {
	ctx := DefaultContext()
	cursor, err := app.GetDiscountsCollection().Find(ctx, bson.M{})
	if err != nil {
		return []Discount{}, app.LogError("GetAllDiscounts", err)
	}
	var discounts []Discount
	if err = cursor.All(ctx, &discounts); err != nil {
		return []Discount{}, app.LogError("GetAllDiscounts", err)
	}
	return discounts, nil
}

// AddDiscounts adds discounts to database
func (app *TasteBuddyApp) AddDiscounts(discounts []Discount) error {
	app.Log("AddDiscounts", "Add "+strconv.Itoa(len(discounts))+" discounts to database")

	// Insert all markets
	opts := options.Update().SetUpsert(true)
	collection := app.GetDiscountsCollection()
	for _, discount := range discounts {

		// Upsert market
		ctx := DefaultContext()
		if _, err := collection.UpdateOne(ctx, bson.M{"internalMarketId": discount.InternalMarketID, "title": discount.Title}, bson.D{{Key: "$set", Value: discount}}, opts); err != nil {
			return app.LogError("AddDiscounts + "+discount.ID.Hex(), err)
		}
	}

	return nil
}

// GetDiscountsByCityFromAPI gets all discounts for a city from the market's APIs
func (app *TasteBuddyApp) GetDiscountsByCityFromAPI(city string) []Discount {
	if markets, err := app.GetMarketsByCity(city); err != nil {
		app.LogError("GetDiscountsByCityFromAPI", err)
		return []Discount{}
	} else {
		var discounts []Discount
		for _, market := range markets {
			discountsForMarket, err := app.GetDiscountsFromAPI(&market)
			if err != nil {
				app.LogError("GetDiscountsByCityFromAPI + "+city, err)
				continue
			}
			discounts = append(discounts, discountsForMarket...)
		}
		return discounts
	}
}

// GoRoutineSaveDiscountsToDB save discounts from different cities to the database
// Is Goroutine
func GoRoutineSaveDiscountsToDB(app *TasteBuddyApp) {
	cities := []string{
		"Konstanz",
		"Berlin",
		"Hamburg",
		"Muenchen",
	}

	app.Log("GoRoutineSaveDiscountsToDB", "Start saving discounts to database")
	for _, city := range cities {
		go app.SaveDiscountsFromAPI(city)
	}
}

// SaveDiscountsFromAPI saves discounts to database
func (app *TasteBuddyApp) SaveDiscountsFromAPI(city string) {
	app.Log("SaveDiscountsFromAPI", "Save discounts for "+city)
	discounts := app.GetDiscountsByCityFromAPI(city)
	if err := app.AddDiscounts(discounts); err != nil {
		app.LogError("SaveDiscountsFromAPI + "+city, err)
		return
	}
	app.Log("SaveDiscountsFromAPI + "+city, "Done saving discounts")
}
