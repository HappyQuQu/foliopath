import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";

import styles from "./Menu.module.css";

export interface MenuAction {
  id: string;
  label: string;
  disabled?: boolean;
  danger?: boolean;
  separatorBefore?: boolean;
  icon?: ReactNode;
  href?: string;
  onSelect: () => void;
}

const enabledItemSelector = '[role="menuitem"]:not(:disabled):not([aria-disabled="true"])';

export interface MenuProps {
  label: string;
  heading?: ReactNode;
  items: MenuAction[];
  anchor: HTMLElement | { x: number; y: number };
  returnFocusTo: HTMLElement | null;
  align?: "start" | "end";
  onClose: () => void;
}

export function Menu({ label, heading, items, anchor, returnFocusTo, align = "start", onClose }: MenuProps) {
  const ref = useRef<HTMLDivElement>(null);
  const closeRef = useRef(onClose);
  closeRef.current = onClose;
  const [position, setPosition] = useState({ left: 0, top: 0 });

  function close(restoreFocus = true) {
    closeRef.current();
    if (restoreFocus && returnFocusTo?.isConnected) returnFocusTo.focus();
  }

  useLayoutEffect(() => {
    const menu = ref.current;
    if (!menu) return;
    const bounds = menu.getBoundingClientRect();
    const source = anchor instanceof HTMLElement ? anchor.getBoundingClientRect() : null;
    const left = source ? (align === "end" ? source.right - bounds.width : source.left) : (anchor as { x: number }).x;
    const top = source?.bottom ?? (anchor as { y: number }).y;
    setPosition({
      left: Math.max(0, Math.min(left, window.innerWidth - bounds.width)),
      top: Math.max(0, Math.min(top, window.innerHeight - bounds.height)),
    });
    menu.querySelector<HTMLElement>(enabledItemSelector)?.focus();
  }, [anchor, align]);

  useEffect(() => {
    function outside(event: PointerEvent) {
      if (event.target instanceof Node && !ref.current?.contains(event.target) && event.target !== returnFocusTo && !returnFocusTo?.contains(event.target)) {
        closeRef.current();
      }
    }
    function resize() { closeRef.current(); }
    document.addEventListener("pointerdown", outside);
    window.addEventListener("resize", resize);
    return () => {
      document.removeEventListener("pointerdown", outside);
      window.removeEventListener("resize", resize);
    };
  }, [returnFocusTo]);

  return createPortal(
    <div
      aria-label={label}
      className={styles.menu}
      ref={ref}
      role="menu"
      style={position}
      onBlur={event => { if (!event.currentTarget.contains(event.relatedTarget) && event.relatedTarget !== returnFocusTo) closeRef.current(); }}
      onKeyDown={event => {
        if (event.key === "Escape" || event.key === "Tab") {
          if (event.key === "Escape") { event.preventDefault(); event.stopPropagation(); }
          close();
          return;
        }
        if (!["ArrowDown", "ArrowUp", "Home", "End"].includes(event.key)) return;
        event.preventDefault();
        const options = Array.from(event.currentTarget.querySelectorAll<HTMLElement>(enabledItemSelector));
        const current = options.indexOf(document.activeElement as HTMLElement);
        const next = event.key === "Home" ? 0 : event.key === "End" ? options.length - 1 : (current + (event.key === "ArrowDown" ? 1 : -1) + options.length) % options.length;
        options[next]?.focus();
      }}
    >
      {heading && <div className={styles.heading} role="presentation">{heading}</div>}
      {items.map(item => {
        const content = <>{item.icon}<span>{item.label}</span></>;
        const attributes = {
          className: styles.item,
          "data-danger": item.danger || undefined,
          "aria-disabled": item.disabled || undefined,
          role: "menuitem",
          tabIndex: -1,
          onClick: (event: React.MouseEvent<HTMLElement>) => {
            if (item.disabled) { event.preventDefault(); return; }
            if (item.href && (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)) { close(false); return; }
            if (item.href) event.preventDefault();
            close();
            item.onSelect();
          },
        };
        return <div key={item.id} role="presentation">
          {item.separatorBefore && <div className={styles.separator} role="separator" />}
          {item.href ? <a {...attributes} href={item.href}>{content}</a> : <button {...attributes} disabled={item.disabled} type="button">{content}</button>}
        </div>;
      })}
    </div>, document.body,
  );
}
