/*
Copyright © 2023 JOSEF MUELLER
*/
package main

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type Recipe struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name" binding:"required"`
	Author      string             `json:"author" bson:"author" binding:"required"`
	Description string             `json:"description" bson:"description" binding:"required"`
	Items       []StepItem         `json:"items,omitempty" bson:"items,omitempty"`
	Steps       []Step             `json:"steps" bson:"steps" binding:"required"`
	Props       struct {
		Url       string    `json:"url,omitempty" bson:"url,omitempty"`
		ImgUrl    string    `json:"imgUrl,omitempty" bson:"imgUrl,omitempty"`
		Duration  int       `json:"duration,omitempty" bson:"duration,omitempty"`
		CreatedAt time.Time `json:"createdAt,omitempty" bson:"createdAt,omitempty"`
		Tags      []string  `json:"tags,omitempty" bson:"tags,omitempty"`
		Likes     int       `json:"likes,omitempty" bson:"likes,omitempty"`
	} `json:"props,omitempty" bson:"props,omitempty"`
	Deleted bool `json:"-" bson:"deleted,omitempty"`
}

type Step struct {
	Description               string                     `json:"description" bson:"description" binding:"required"`
	Items                     []StepItem                 `json:"items,omitempty" bson:"items,omitempty"`
	ImgUrl                    string                     `json:"imgUrl,omitempty" bson:"imgUrl,omitempty"`
	Duration                  int                        `json:"duration,omitempty" bson:"duration,omitempty"`
	AdditionalStepInformation *AdditionalStepInformation `json:"additional,omitempty" bson:"additional,omitempty"`
}

func StepFromDescription(description string) Step {
	step := Step{}
	step.Description = description
	step.Items = []StepItem{}
	step.Duration = 0
	step.ImgUrl = ""
	step.AdditionalStepInformation = nil
	return step
}

type AdditionalStepInformation struct {
	Type                  string `json:"informationType,omitempty" bson:"informationType,omitempty"`
	BakingStepInformation `json:",inline,omitempty" bson:",inline,omitempty"`
}

type BakingStepInformation struct {
	Temperature int    `json:"temperature,omitempty" bson:"temperature,omitempty"`
	BakingType  string `json:"bakingType,omitempty" bson:"bakingType,omitempty"`
}

type StepItem struct {
	ItemID primitive.ObjectID `json:"-" bson:"_id,omitempty"`
	Amount int                `json:"amount" bson:"amount" binding:"required"`
	Unit   string             `json:"unit,omitempty" bson:"unit,omitempty" binding:"required"`
	Item   Item               `json:"item" bson:"-" binding:"required"`
}

type Item struct {
	ID     primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Name   string             `json:"name" bson:"name" binding:"required"`
	Type   string             `json:"type,omitempty" bson:"type,omitempty"`
	ImgUrl string             `json:"imgUrl,omitempty" bson:"imgUrl,omitempty"`
}

// HandleGetAllRecipes gets called by router
// Calls getRecipesFromDB and handles the context
func (server *TasteBuddyServer) HandleGetAllRecipes(context *gin.Context) {
	recipes, err := server.GetAllRecipes()
	if err != nil {
		server.LogError("HandleGetAllRecipes", err)
		ServerError(context, true)
		return
	}
	Success(context, recipes)
}

func (server *TasteBuddyServer) HandleGetRecipeById(context *gin.Context) {
	id := context.Param("id")

	// convert id to primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		server.LogError("HandleGetRecipeById", err)
		ServerError(context, true)
		return
	}

	recipe, err := server.GetRecipeById(objectID)
	if err != nil {
		server.LogError("HandleGetRecipeById", err)
		ServerError(context, true)
		return
	}
	Success(context, recipe)
}

// HandleGetRandomRecipe gets called by router
// Calls getRecipesFromDB and selects a random recipe
func (server *TasteBuddyServer) HandleGetRandomRecipe(context *gin.Context) {
	recipes, err := server.GetAllRecipes()
	if err != nil {
		server.LogError("HandleGetRandomRecipe", err)
		ServerError(context, true)
		return
	}

	// check if there are recipes
	if len(recipes) == 0 {
		NotFoundError(context, "Recipes")
		return
	}

	// get random recipe
	randomIndex := rand.Intn(len(recipes))
	Success(context, recipes[randomIndex])
}

// HandleAddRecipe gets called by router
// Calls addRecipeToDB and handles the context
func (server *TasteBuddyServer) HandleAddRecipe(context *gin.Context) {
	server.LogContextHandle(context, "HandleAddRecipe", "Trying to add/update recipe")

	// try to bind json to recipe
	var newRecipe Recipe
	if err := context.BindJSON(&newRecipe); err != nil {
		server.LogError("HandleAddRecipe", err)
		BadRequestError(context, "Invalid Recipe")
		return
	}

	var recipeId primitive.ObjectID
	var err error
	if recipeId, err = server.AddOrUpdateRecipe(newRecipe); err != nil {
		server.LogError("HandleAddRecipe", err)
		ServerError(context, true)
		return
	}
	server.LogContextHandle(context, "HandleAddRecipe", "Added/Updated recipe "+newRecipe.Name+" ("+newRecipe.ID.Hex()+")")
	Success(context, "Saved recipe "+recipeId.Hex())
}

// HandleDeleteRecipeById gets called by router
// Calls DeleteRecipeById and handles the context
func (server *TasteBuddyServer) HandleDeleteRecipeById(context *gin.Context) {
	id := context.Param("id")
	server.LogContextHandle(context, "HandleDeleteRecipeById", "Trying to delete recipe "+id)

	// convert id to primitive.ObjectID
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		server.LogError("HandleDeleteRecipeById", err)
		ServerError(context, true)
		return
	}

	// delete recipe
	if _, err := server.DeleteRecipeById(objectID); err != nil {
		server.LogError("HandleDeleteRecipeById", err)
		ServerError(context, true)
		return
	}
	server.LogContextHandle(context, "HandleDeleteRecipeById", "Deleted recipe "+id)
	Success(context, "Deleted recipe "+id)
}

// HandleFindRecipesByItemNames gets called by router
// Calls GetRecipesByItemNames and handles the context
func (server *TasteBuddyServer) HandleFindRecipesByItemNames(context *gin.Context) {
	itemIds := context.Param("itemIds")

	// split itemIds string into array
	splitItemIds := strings.Split(itemIds, ",")

	recipes, err := server.GetRecipesByItemNames(splitItemIds)
	if err != nil {
		ServerError(context, true)
		server.LogError("HandleFindRecipesByItemNames", err)
	}
	Success(context, recipes)
}

// GetRecipesCollection gets recipes collection from database
func (app *TasteBuddyApp) GetRecipesCollection() *mongo.Collection {
	return app.client.Database("tastebuddy").Collection("recipes")
}

// GetAllRecipes gets all recipes from database
func (app *TasteBuddyApp) GetAllRecipes() ([]Recipe, error) {
	ctx := DefaultContext()

	// get all recipes from database that are not deleted
	cursor, err := app.GetRecipesCollection().Find(ctx, bson.M{"deleted": bson.M{"$ne": true}})
	if err != nil {
		return []Recipe{}, app.LogError("GetAllRecipes", err)
	}

	// try to get all recipes from database and bind them to recipesFromDatabase
	var recipesFromDatabase []Recipe
	if err = cursor.All(ctx, &recipesFromDatabase); err != nil {
		return []Recipe{}, app.LogError("GetAllRecipes", err)
	}

	if recipesFromDatabase == nil {
		// replace nil with empty array
		recipesFromDatabase = []Recipe{}
	}

	// get all items from database
	items, err := app.GetAllItems()
	if err != nil {
		return []Recipe{}, app.LogError("GetAllRecipes", err)
	}

	// prepare items for each recipe
	for i := range recipesFromDatabase {
		recipesFromDatabase[i].MapItemIdsToItem(items)
	}

	return recipesFromDatabase, nil
}

func (recipe *Recipe) MapItemIdsToItem(items []Item) {
	// create map of items
	itemsMap := make(map[primitive.ObjectID]Item)
	for i := range items {
		itemsMap[items[i].ID] = items[i]
	}

	// replace item ids with items
	for i := range recipe.Steps {
		for j := range recipe.Steps[i].Items {
			recipe.Steps[i].Items[j].Item = itemsMap[recipe.Steps[i].Items[j].ItemID]
		}
	}
}

func (app *TasteBuddyApp) GetRecipeById(id primitive.ObjectID) (Recipe, error) {
	ctx := DefaultContext()
	// try to get collection of recipes
	recipe := app.GetRecipesCollection().FindOne(ctx, bson.M{"_id": id})

	if recipe.Err() != nil {
		return Recipe{}, app.LogError("GetRecipeById", recipe.Err())
	}

	var recipeFromDatabase Recipe
	if err := recipe.Decode(&recipeFromDatabase); err != nil {
		return Recipe{}, app.LogError("GetRecipeById", err)
	}

	return recipeFromDatabase, nil
}

// AddOrUpdateRecipe adds a new recipe to the database of recipes
// and returns all the id of the new recipe or the old recipe
func (app *TasteBuddyApp) AddOrUpdateRecipe(newRecipe Recipe) (primitive.ObjectID, error) {
	ctx := DefaultContext()
	var err error
	var objectId primitive.ObjectID

	// get the items from the recipe and add them to the database
	if err = app.PrepareRecipeForDB(newRecipe); err != nil {
		return objectId, app.LogError("AddOrUpdateRecipe + recipe "+newRecipe.Name, err)
	}

	if newRecipe.ID.IsZero() {
		// add new recipe
		// set createdAt to current time
		app.LogWarning("AddOrUpdateRecipe + recipe "+newRecipe.Name, "Add new recipe to database")
		var result *mongo.InsertOneResult
		newRecipe.Props.CreatedAt = time.Now()
		result, err = app.GetRecipesCollection().InsertOne(ctx, newRecipe)
		objectId = result.InsertedID.(primitive.ObjectID)
	} else {
		// update recipe
		app.LogWarning("AddOrUpdateRecipe + recipe "+newRecipe.Name+"("+newRecipe.ID.Hex()+")", "Update existing recipe in database")
		_, err = app.GetRecipesCollection().UpdateOne(ctx,
			bson.D{{Key: "_id", Value: newRecipe.ID}},
			bson.D{{Key: "$set", Value: newRecipe}})
		objectId = newRecipe.ID
	}
	if err != nil {
		return objectId, app.LogError("AddOrUpdateRecipe + recipe "+newRecipe.Name+"("+objectId.Hex()+")", err)
	}

	app.LogWarning("AddOrUpdateRecipe + recipe "+newRecipe.Name+"("+objectId.Hex()+")", "Successful operation")
	return objectId, nil
}

func (app *TasteBuddyApp) DeleteRecipeById(id primitive.ObjectID) (primitive.ObjectID, error) {
	ctx := DefaultContext()

	// delete recipe by setting deleted to true
	app.LogWarning("DeleteRecipeById", "Delete recipe "+id.Hex()+" from database")
	if _, err := app.GetRecipesCollection().UpdateByID(ctx, id, bson.D{{Key: "$set", Value: bson.D{{Key: "deleted", Value: true}}}}); err != nil {
		return id, app.LogError("DeleteRecipeById", err)
	}

	return id, nil
}

func (app *TasteBuddyApp) PrepareRecipeForDB(recipe Recipe) error {
	// normalize all items in recipe
	for stepIndex, step := range recipe.Steps {
		for itemIndex, stepItem := range step.Items {
			var err error

			// if item in stepItem has an id, it is already in the database
			itemId, err := app.AddOrUpdateItem(stepItem.Item)
			if err != nil {
				return app.LogError("PrepareForDB + "+recipe.Name+" + "+stepItem.Item.Name, err)
			}
			app.Log("PrepareForDB + recipe "+recipe.Name+"("+recipe.ID.Hex()+")", "Map "+stepItem.Item.Name+" to "+itemId.Hex())
			stepItem.ItemID = itemId
			stepItem.Item = Item{}
			recipe.Steps[stepIndex].Items[itemIndex] = stepItem
		}
	}
	return nil
}

// GetRecipesByItemNames gets the recipes in which the given items are used
func (app *TasteBuddyApp) GetRecipesByItemNames(splitItemIds []string) ([]Recipe, error) {
	// use map since its easier to avoid duplicates
	var recipesMap = make(map[string]Recipe)

	// get all recipes from database
	recipes, err := app.GetAllRecipes()
	if err != nil {
		return []Recipe{}, app.LogError("GetRecipesByItemNames", err)
	}

	for _, itemID := range splitItemIds {
		for _, recipe := range recipes {
			// iterate through each item used in recipe
			var itemsByRecipe = recipe.ExtractItems()
			for _, recipeItem := range itemsByRecipe {
				// add recipe to map if not already added and if itemID corresponds to recipeItem.ID
				if _, ok := recipesMap[recipe.ID.Hex()]; !ok && itemID == recipeItem.ID.Hex() {
					recipesMap[recipe.ID.Hex()] = recipe
				}
			}
		}
	}

	// convert map to array
	var filteredRecipes []Recipe
	for _, recipe := range recipesMap {
		filteredRecipes = append(filteredRecipes, recipe)
	}

	return filteredRecipes, nil
}

// CleanUpItemsInRecipes removes all items from recipes that are not in the database
// and replaces them with the best item from the database
func (app *TasteBuddyApp) CleanUpItemsInRecipes() error {
	var err error

	var items []Item
	var recipes []Recipe

	// get items
	items, err = app.GetAllItems()
	if err != nil {
		return app.LogError("CleanUpItemsInRecipes", err)
	}

	// create map of the "best" item for each name
	var itemMap = make(map[string]Item)
	for _, item := range items {
		// check if map already contains item
		if itemFromMap, ok := itemMap[item.Name]; ok {
			// check if item is better than item in map
			if item.GetItemQuality() > itemFromMap.GetItemQuality() {
				// replace item in map
				itemMap[item.Name] = item
				app.Log("CleanUpItemsInRecipes", fmt.Sprintf("replace %s with %s", itemFromMap.ID.Hex(), item.ID.Hex()))
			}
		} else {
			// add item to map
			itemMap[item.Name] = item
		}
	}

	// get recipes
	recipes, err = app.GetAllRecipes()
	if err != nil {
		return app.LogError("CleanUpItemsInRecipes", err)
	}

	// go through all recipes
	var amountCleanedUp = 0
	for _, recipe := range recipes {
		for stepIndex, step := range recipe.Steps {
			for itemIndex, stepItem := range step.Items {
				// check if item is in map
				if item, ok := itemMap[stepItem.Item.Name]; ok {
					// replace item
					stepItem.Item = item
					stepItem.ItemID = item.ID
					recipe.Steps[stepIndex].Items[itemIndex] = stepItem
					amountCleanedUp++
				}
			}
		}
	}

	// update recipes
	for _, recipe := range recipes {
		if _, err = app.AddOrUpdateRecipe(recipe); err != nil {
			return app.LogError("CleanUpItemsInRecipes", err)
		}
	}

	app.Log("CleanUpItemsInRecipes", fmt.Sprintf("Clean up %d recipes...", amountCleanedUp))

	return nil
}

// CleanUpUnusedAttributesInRecipes marshals and unmarshal all recipes and
// tries to remove all unused attributes
func (app *TasteBuddyApp) CleanUpUnusedAttributesInRecipes() error {
	recipes, err := app.GetAllRecipes()
	if err != nil {
		return app.LogError("CleanUpUnusedAttributesInRecipes", err)
	}

	for _, recipe := range recipes {
		if _, err := app.AddOrUpdateRecipe(recipe); err != nil {
			return app.LogError("CleanUpUnusedAttributesInRecipes", err)
		}
	}

	return nil
}

// GoRoutineCleanUpRecipes contains goroutines that are called every 6 hours
// to clean up parts of the recipes
func GoRoutineCleanUpRecipes(app *TasteBuddyApp) {
	for {
		app.CleanUpItemsInRecipes()
		app.CleanUpUnusedAttributesInRecipes()
		// clean up recipes every 6 hours
		time.Sleep(6 * time.Hour)
	}
}
