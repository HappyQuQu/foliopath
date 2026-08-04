import { Fragment, type ReactNode } from "react";

type Block =
  | { kind: "heading"; text: string }
  | { kind: "list"; items: string[] }
  | { kind: "paragraph"; text: string };

const markdownLink = /!?\[([^\]]*)\]\([^\s)]+\)/g;
const trailingCommitReference = /\s+\([0-9a-f]{7,40}\)$/i;

export function releaseNoteBlocks(notes: string, version: string): Block[] {
  const blocks: Block[] = [];
  let list: string[] = [];
  let paragraph: string[] = [];

  function flushList() {
    if (list.length > 0) blocks.push({ kind: "list", items: list });
    list = [];
  }

  function flushParagraph() {
    if (paragraph.length > 0) {
      blocks.push({ kind: "paragraph", text: paragraph.join(" ") });
    }
    paragraph = [];
  }

  for (const rawLine of notes.replaceAll("\r\n", "\n").split("\n")) {
    const line = rawLine.trim();
    if (!line) {
      flushList();
      flushParagraph();
      continue;
    }

    const heading = line.match(/^#{1,6}\s+(.+)$/);
    if (heading) {
      flushList();
      flushParagraph();
      const text = plainText(heading[1] ?? "");
      if (!isReleaseTitle(text, version)) blocks.push({ kind: "heading", text });
      continue;
    }

    const item = line.match(/^(?:[-*+]\s+|\d+\.\s+)(.+)$/);
    if (item) {
      flushParagraph();
      list.push(plainText(item[1] ?? ""));
      continue;
    }

    flushList();
    paragraph.push(plainText(line));
  }
  flushList();
  flushParagraph();
  return blocks.filter((block) => block.kind !== "list" || block.items.some(Boolean));
}

function plainText(value: string): string {
  return value
    .replace(markdownLink, "$1")
    .replace(trailingCommitReference, "")
    .replace(/[`*_~]/g, "")
    .trim();
}

function isReleaseTitle(value: string, version: string): boolean {
  const normalized = version.replace(/^v/, "");
  return value.startsWith(version) || value.startsWith(normalized);
}

export function ReleaseNotes({ notes, version }: { notes: string; version: string }) {
  const blocks = releaseNoteBlocks(notes, version);
  if (blocks.length === 0) return null;

  const content: ReactNode[] = blocks.map((block, index) => {
    const key = `${block.kind}-${index}`;
    if (block.kind === "heading") return <h3 key={key}>{block.text}</h3>;
    if (block.kind === "list") {
      return (
        <ul key={key}>
          {block.items.filter(Boolean).map((item, itemIndex) => (
            <li key={`${item}-${itemIndex}`}>{item}</li>
          ))}
        </ul>
      );
    }
    return <p key={key}>{block.text}</p>;
  });

  return <Fragment>{content}</Fragment>;
}
