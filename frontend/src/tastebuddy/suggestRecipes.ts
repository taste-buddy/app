import {useRecipeStore} from "@/storage";
import {logDebug, Recipe, StepItem} from "@/tastebuddy";

class RecipeSuggestion {

    recipe_id: string;
    recipe?: Recipe;
    recipe_price?: number;
    missing_items?: {
        item: StepItem;
        price?: number;
    }[]

    constructor() {
        this.recipe_id = ''
        this.recipe_price = 0
        this.missing_items = []
    }


    /**
     * Get the recipe of the suggestion
     */
    public getRecipe(): Recipe {
        return this.recipe ?? new Recipe()
    }

    public getMissingItems(): StepItem[] {
        return this.missing_items?.map(missing_item => missing_item.item ?? new StepItem()) ?? []
    }
}

type ItemQuery = {
    id?: string,
    exclude?: boolean,
}

type SearchQuery = {
    items: ItemQuery[],
    tags: string[]
    price?: number,
    duration?: number,
}

class SearchQueryBuilder {
    private readonly items: ItemQuery[]
    private tags: string[]
    private price: number | undefined
    private duration: number | undefined
    private city?: string

    constructor() {
        this.items = []
        this.tags = []
        this.price = 0
        this.duration = 0
    }

    public addItem(item: ItemQuery): this {
        this.items.push(item)
        return this
    }

    public setItemIds(itemIds: {
        [id: string]: boolean
    }): this {
        const items = Object.entries(itemIds).map(([id, include]: [string, boolean]) => ({
            id,
            name: '',
            exclude: !include
        }))
        this.items.push(...items)
        return this
    }

    public setItems(items: ItemQuery[]): this {
        this.items.push(...items)
        return this
    }

    public setTags(tags: string[]): this {
        this.tags = tags
        return this
    }

    public setPrice(price?: number): this {
        this.price = price
        return this
    }

    public setDuration(duration?: number): this {
        this.duration = duration
        return this
    }

    public setCity(city: string): this {
        this.city = city
        return this
    }

    public build(): SearchQuery {
        return {
            items: this.items,
            tags: this.tags,
            price: this.price,
            duration: this.duration
        }
    }
}

/**
 * Search recipes based on the given query
 * @param query
 */
function suggestRecipes(query: SearchQuery): RecipeSuggestion[] {
    const store = useRecipeStore()
    const recipes: Recipe[] = store.getRecipesAsList

    const suggestedRecipes = recipes.filter((recipe: Recipe) => {
        return filterRecipeByItems(recipe, query.items) &&
            filterRecipeByDuration(recipe, query.duration) &&
            filterRecipeByTag(recipe, query.tags) &&
            filterByPrice(recipe, query.price)
    })

    return suggestedRecipes.map((recipe: Recipe) => {
        const suggestion = new RecipeSuggestion()
        suggestion.recipe = recipe
        suggestion.recipe_price = recipe.getPrice()
        suggestion.missing_items = []
        return suggestion
    })
}

/**
 * Checks if a recipe contains all items in the itemQuery
 * @param itemQuery
 * @param recipe
 * @return {boolean} true if the itemQuery is satisfied by the recipe
 */
function filterRecipeByItems(recipe: Recipe, itemQuery: ItemQuery[]): boolean {
    const success = itemQuery.every((itemQ: ItemQuery) => {
        // Check if item exists in recipe
        const itemExists = recipe.hasItem(itemQ.id)
        // Either item exists and we want to include it,
        // or item doesn't exist, and we want to exclude it
        return itemExists !== itemQ.exclude
    })
    logDebug('filterRecipeByItems', recipe, itemQuery, success)
    return success
}

/**
 * Checks if a recipe is within the maxDuration
 * @param recipe
 * @param maxDuration
 */
function filterRecipeByDuration(recipe: Recipe, maxDuration?: number): boolean {
    if (maxDuration === undefined) {
        return true
    }
    const success = recipe.getDuration() <= maxDuration
    logDebug('filterRecipeByDuration', recipe.getDuration(), maxDuration, success)
    return success
}

/**
 * Checks if a recipe contains all tags in the tagQuery
 * @param recipe
 * @param tags
 */
function filterRecipeByTag(recipe: Recipe, tags: string[]): boolean {
    const recipeTags = recipe.getTags()
    const success = tags.every((tag: string) => recipeTags.includes(tag))
    logDebug('filterRecipeByTag', recipeTags, tags, success)
    return success
}

/**
 * Checks if a recipe is within the price range
 */
function filterByPrice(recipe: Recipe, maxPrice?: number): boolean {
    if (maxPrice === undefined) {
        return true
    }
    const recipePrice = recipe.getPrice()
    const success = recipePrice <= maxPrice
    console.log('filterByPrice', recipePrice, maxPrice, success)
    return success
}

export {
    ItemQuery,
    suggestRecipes,
    RecipeSuggestion,
    SearchQueryBuilder
}