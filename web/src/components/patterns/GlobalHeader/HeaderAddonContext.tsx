import { createContext, useContext, type ReactNode } from "react";

const HeaderAddonContext = createContext<ReactNode>(null);

export function HeaderAddonProvider({
  children,
  value,
}: {
  children: ReactNode;
  value: ReactNode;
}) {
  return (
    <HeaderAddonContext.Provider value={value}>
      {children}
    </HeaderAddonContext.Provider>
  );
}

export function useHeaderAddon() {
  return useContext(HeaderAddonContext);
}
