import type { LucideIcon } from "lucide-react";
import {
  Activity,
  Apple,
  ArrowLeftRight,
  Baby,
  Banknote,
  Beer,
  Bike,
  BookOpen,
  Briefcase,
  Bus,
  Car,
  CircleDashed,
  CircleParking,
  Coffee,
  Coins,
  CreditCard,
  Droplet,
  Dumbbell,
  Film,
  Flame,
  Fuel,
  Gamepad2,
  Gift,
  GraduationCap,
  HandCoins,
  Heart,
  House,
  Landmark,
  Music,
  PawPrint,
  Phone,
  PiggyBank,
  Plane,
  Receipt,
  Repeat,
  Send,
  ShoppingBag,
  ShoppingCart,
  Split,
  Train,
  Trash2,
  TrendingUp,
  Utensils,
  Vault,
  Wallet,
  Wifi,
  Wrench,
  Zap,
} from "lucide-react";

/** One selectable icon: a stable kebab-case token (what gets stored) and
 *  its statically imported `lucide-react` component. */
export type EntityIconEntry = { token: string; Component: LucideIcon };

/** A named group of icons in the picker. `headingKey` is an i18n key. */
export type EntityIconGroup = { headingKey: string; icons: EntityIconEntry[] };

/**
 * The curated icon set offered for accounts and categories. Every component
 * is imported by name above so the bundle carries only these icons — never
 * a wildcard or runtime-dynamic import. This list is also the allow-list:
 * a stored token not present here renders no glyph (see `EntityIcon`).
 */
export const ENTITY_ICON_GROUPS: EntityIconGroup[] = [
  {
    headingKey: "entityIcons.groups.banking",
    icons: [
      { token: "wallet", Component: Wallet },
      { token: "piggy-bank", Component: PiggyBank },
      { token: "landmark", Component: Landmark },
      { token: "credit-card", Component: CreditCard },
      { token: "banknote", Component: Banknote },
      { token: "coins", Component: Coins },
      { token: "vault", Component: Vault },
    ],
  },
  {
    headingKey: "entityIcons.groups.income",
    icons: [
      { token: "briefcase", Component: Briefcase },
      { token: "hand-coins", Component: HandCoins },
      { token: "trending-up", Component: TrendingUp },
      { token: "gift", Component: Gift },
      { token: "receipt", Component: Receipt },
    ],
  },
  {
    headingKey: "entityIcons.groups.homeBills",
    icons: [
      { token: "house", Component: House },
      { token: "zap", Component: Zap },
      { token: "droplet", Component: Droplet },
      { token: "flame", Component: Flame },
      { token: "wifi", Component: Wifi },
      { token: "phone", Component: Phone },
      { token: "wrench", Component: Wrench },
      { token: "trash-2", Component: Trash2 },
    ],
  },
  {
    headingKey: "entityIcons.groups.food",
    icons: [
      { token: "shopping-cart", Component: ShoppingCart },
      { token: "shopping-bag", Component: ShoppingBag },
      { token: "utensils", Component: Utensils },
      { token: "coffee", Component: Coffee },
      { token: "apple", Component: Apple },
      { token: "beer", Component: Beer },
    ],
  },
  {
    headingKey: "entityIcons.groups.transport",
    icons: [
      { token: "car", Component: Car },
      { token: "bus", Component: Bus },
      { token: "train", Component: Train },
      { token: "plane", Component: Plane },
      { token: "bike", Component: Bike },
      { token: "fuel", Component: Fuel },
      { token: "circle-parking", Component: CircleParking },
    ],
  },
  {
    headingKey: "entityIcons.groups.life",
    icons: [
      { token: "heart", Component: Heart },
      { token: "activity", Component: Activity },
      { token: "gamepad-2", Component: Gamepad2 },
      { token: "film", Component: Film },
      { token: "music", Component: Music },
      { token: "dumbbell", Component: Dumbbell },
      { token: "book-open", Component: BookOpen },
      { token: "graduation-cap", Component: GraduationCap },
      { token: "paw-print", Component: PawPrint },
      { token: "baby", Component: Baby },
    ],
  },
  {
    headingKey: "entityIcons.groups.transfers",
    icons: [
      { token: "arrow-left-right", Component: ArrowLeftRight },
      { token: "repeat", Component: Repeat },
      { token: "split", Component: Split },
      { token: "send", Component: Send },
      { token: "circle-dashed", Component: CircleDashed },
    ],
  },
];

const ICON_BY_TOKEN: Map<string, LucideIcon> = new Map(
  ENTITY_ICON_GROUPS.flatMap((g) => g.icons).map((i) => [i.token, i.Component]),
);

/** The `lucide-react` component for a stored icon token, or `undefined`
 *  when the token is empty or not in the curated set. */
export function entityIconComponent(
  token: string | undefined,
): LucideIcon | undefined {
  if (!token) {
    return undefined;
  }
  return ICON_BY_TOKEN.get(token);
}
