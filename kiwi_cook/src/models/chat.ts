import { Recipe } from 'src/models/recipe.ts';

export type MessageType = 'text' | 'image' | 'recipe' | 'options' | 'multiOptions' | 'slider';

export interface BaseMessage {
  id: number;
  sender: string;
  sent: boolean;
  type: MessageType;
}

export interface TextMessage extends BaseMessage {
  type: 'text';
  content: string;
}

export interface ImageMessage extends BaseMessage {
  type: 'image';
  content: string;
}

export interface RecipeMessage extends BaseMessage {
  type: 'recipe';
  content: Recipe[];
}

export interface OptionsMessage extends BaseMessage {
  type: 'options' | 'multiOptions';
  content: string[];
}

export interface SliderMessage extends BaseMessage {
  type: 'slider';
  content: {
    label: string;
    value: number;
    min: number;
    max: number;
    step: number;
  };
}

export type Message = TextMessage | ImageMessage | RecipeMessage | OptionsMessage | SliderMessage;

export type KiwiMessageState =
  'start'
  | 'recipeType'
  | 'dietaryRestrictions'
  | 'cookingTime'
  | 'cuisine'
  | 'searching'
  | 'displayingResults';
