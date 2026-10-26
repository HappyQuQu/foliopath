import { useSearchParams } from "react-router-dom";

// Shared owner for the directory and its contextual sidebar. Named people are global.
export function usePeopleDirectoryState() {
  const [params, setParams] = useSearchParams();
  const search = params.get("q") ?? "";
  function setSearch(value: string) {
    const next = new URLSearchParams(params); next.delete("peopleCursor");
    if (value) next.set("q", value); else next.delete("q");
    setParams(next, { replace: true });
  }
  return { search, cursor: params.get("peopleCursor") ?? "", setSearch };
}
