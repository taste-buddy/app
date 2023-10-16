import {createRouter} from '@ionic/vue-router';
import {createWebHashHistory, Router, RouteRecordRaw} from 'vue-router';
import TabsPage from '@/app/views/VTabs.vue'
import VRecipe from "@/shared/views/VRecipe.vue";
import VRecipeSuggestions from "@/app/views/recipe/VRecipeSuggestions.vue";
import {useRecipeStore} from "@/app/storage";

const routes: Array<RouteRecordRaw> = [
    {
        name: 'Home',
        path: '/',
        component: TabsPage,
        redirect: () => ({name: 'RecipeSuggestions'}),
        children: [
            // Recipes
            {
                name: 'Recipe',
                path: 'recipe/show/:id',
                component: VRecipe
            },
            {
                name: 'RecipeOfTheDay',
                path: 'recipe/of-the-day',
                redirect() {
                    const recipeStore = useRecipeStore()
                    const recipe = recipeStore.getRecipeOfTheDay
                    if (recipe) {
                        return {name: 'Recipe', params: {id: recipe.getId()}}
                    }
                    return {name: 'Home'}
                }
            },
            {
                name: 'RecipeSuggestions',
                path: 'recipe/suggestions',
                component: VRecipeSuggestions
            },
            {
                name: 'SavedRecipes',
                path: 'recipe/saved',
                component: () => import('@/app/views/recipe/VSavedRecipes.vue')
            },
            {
                name: 'Settings',
                path: 'settings',
                component: () => import('@/app/views/VSettings.vue')
            },
            // 404
            {
                name: 'NotFound',
                path: '/:pathMatch(.*)*',
                redirect: () => ({name: 'Home'}),
            }
        ]
    }
]

/**
 * Create router
 * @returns {Router}
 */
export function createTasteBuddyRouter(): Router {
    return createRouter({
        history: createWebHashHistory(process.env.BASE_URL),
        routes
    })
}