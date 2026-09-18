import { Dialog, DialogPanel, DialogTitle } from "@headlessui/react";
import type { ReactNode } from "react";

type ModalSize = "sm" | "md";

type ModalProps = {
  open: boolean;
  /**
   * Called when the visitor dismisses the modal (Escape, backdrop). Still
   * required when `dismissable` is false — the shell simply never calls it.
   */
  onClose: () => void;
  title: ReactNode;
  /** Panel width and density. Default `"sm"`. */
  size?: ModalSize | undefined;
  /**
   * When false, Escape and a backdrop click do nothing — for a modal that
   * must stay open while work is in flight. Default true.
   */
  dismissable?: boolean | undefined;
  children: ReactNode;
};

// Size varies the panel's width only. Gap and padding are deliberately
// identical across sizes: the three panel shapes this replaced differed on
// two axes, and collapsing density to one value is what keeps a new dialog
// from having to pick.
// max-h/overflow so a panel taller than the viewport scrolls instead of
// running off the bottom of a phone screen with its actions out of reach.
const PANEL_BASE =
  "flex w-full max-h-[calc(100dvh_-_2rem)] flex-col gap-4 overflow-y-auto rounded-lg bg-white p-6 dark:bg-neutral-900";

const PANEL_CLASS: Record<ModalSize, string> = {
  sm: `${PANEL_BASE} max-w-sm`,
  md: `${PANEL_BASE} max-w-md`,
};

const noop = () => {};

/**
 * The one modal shell every dialog in the app renders through: the
 * backdrop, the centering wrapper, the panel, and the title. Call sites
 * supply only their content — deliberately no panel `className`
 * passthrough, so every dialog stays identical at a given size and a new
 * appearance means a new named variant here rather than local drift.
 *
 * Every modal sits at the same `z-50`. A modal opened from inside another
 * renders above it because Headless UI portals stack in mount order, so
 * there is no stacking level to configure (see `/categories`, whose delete
 * confirmation opens while its edit dialog is still open).
 */
export function Modal({
  open,
  onClose,
  title,
  size,
  dismissable,
  children,
}: ModalProps) {
  return (
    <Dialog
      open={open}
      onClose={dismissable === false ? noop : onClose}
      className="relative z-50"
    >
      <div className="fixed inset-0 bg-black/40" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className={PANEL_CLASS[size ?? "sm"]}>
          <DialogTitle className="text-base font-semibold">{title}</DialogTitle>
          {children}
        </DialogPanel>
      </div>
    </Dialog>
  );
}
