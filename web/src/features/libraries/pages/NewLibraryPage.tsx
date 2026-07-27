import {
  CaretRight,
  Check,
  Database,
  FolderOpen,
  Info,
} from "@phosphor-icons/react";
import {
  useMemo,
  useRef,
  useState,
  type FormEvent,
} from "react";
import { useNavigate } from "react-router-dom";

import { AppShell } from "../../../components/patterns/AppShell/AppShell";
import {
  Button,
  ErrorState,
  FormField,
  InlineStatus,
  LoadingState,
  useToast,
} from "../../../components/ui";
import type { AuthenticatedSession } from "../../../lib/api/auth";
import { ApiError } from "../../../lib/api/errors";
import type {
  LibraryPathBlockedReason,
  LibraryPathEntry,
} from "../../../lib/api/libraries";
import {
  useLocale,
  type MessageKey,
} from "../../../lib/i18n/LocaleProvider";
import { useSubmissionGuard } from "../../../lib/useSubmissionGuard";
import { paths } from "../../../routes/paths";
import {
  useCreateLibraryMutation,
  useLibraryPathsQuery,
} from "../queries";
import styles from "./NewLibraryPage.module.css";

const steps: MessageKey[] = [
  "newLibrary.stepName",
  "newLibrary.stepPath",
  "newLibrary.stepReview",
];

const blockedReasonMessage: Record<LibraryPathBlockedReason, MessageKey> = {
  overlapping_library: "newLibrary.blockedOverlap",
  ancestor_of_library: "newLibrary.blockedAncestor",
  descendant_of_library: "newLibrary.blockedDescendant",
  unreadable: "newLibrary.blockedUnreadable",
  symlink: "newLibrary.blockedSymlink",
  mount_boundary: "newLibrary.blockedMount",
  unavailable: "newLibrary.blockedUnavailable",
};

export function NewLibraryPage({ session }: { session: AuthenticatedSession }) {
  const { t } = useLocale();
  const toast = useToast();
  const navigate = useNavigate();
  const [step, setStep] = useState(1);
  const [name, setName] = useState("");
  const [nameError, setNameError] = useState<string>();
  const [parent, setParent] = useState("");
  const [selectedPath, setSelectedPath] = useState<string | null>(null);
  const [pageError, setPageError] = useState<string>();
  const requestKey = useRef(crypto.randomUUID());
  const runSubmission = useSubmissionGuard();
  const pathQuery = useLibraryPathsQuery(parent);
  const {
    fetchNextPage: loadNextPathPage,
    refetch: refreshPaths,
  } = pathQuery;
  const createMutation = useCreateLibraryMutation();
  const pathItems = useMemo(
    () => pathQuery.data?.pages.flatMap((page) => page.items) ?? [],
    [pathQuery.data],
  );
  const location = pathQuery.data?.pages[0]?.location;
  const breadcrumbs = pathQuery.data?.pages[0]?.breadcrumbs ?? [];

  function updateName(nextName: string) {
    setName(nextName);
    setNameError(undefined);
    requestKey.current = crypto.randomUUID();
  }

  function updateSelectedPath(nextPath: string) {
    setSelectedPath(nextPath);
    setPageError(undefined);
    requestKey.current = crypto.randomUUID();
  }

  function continueFromName(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const normalized = name.trim();
    if (!normalized) {
      setNameError(t("newLibrary.nameRequired"));
      return;
    }
    if (normalized.length > 128) {
      setNameError(t("newLibrary.nameTooLong"));
      return;
    }
    setName(normalized);
    setStep(2);
  }

  async function submitLibrary() {
    if (selectedPath === null) return;
    setPageError(undefined);

    await runSubmission(async () => {
      try {
        await createMutation.mutateAsync({
          csrfToken: session.csrfToken,
          idempotencyKey: requestKey.current,
          name,
          rootPath: selectedPath,
        });
        toast.show({ message: t("newLibrary.created"), tone: "success" });
        navigate(paths.libraries, { replace: true });
      } catch (error) {
        setPageError(messageForCreateError(error, t));
      }
    });
  }

  return (
    <AppShell
      active="libraries"
      identity={session.administrator.displayName}
      librariesHref={paths.libraries}
      settingsHref={paths.generalSettings}
      title={t("newLibrary.title")}
    >
      <section className={styles.page} aria-labelledby="new-library-title">
        <header className={styles.heading}>
          <div>
            <p className={styles.eyebrow}>{t("newLibrary.eyebrow")}</p>
            <h1 id="new-library-title">{t("newLibrary.title")}</h1>
            <p>{t("newLibrary.description")}</p>
          </div>
          <Button onClick={() => navigate(paths.libraries)} variant="quiet">
            {t("newLibrary.cancel")}
          </Button>
        </header>

        <ol className={styles.stepper}>
          {steps.map((label, index) => {
            const stepNumber = index + 1;
            return (
              <li
                className={step >= stepNumber ? styles.activeStep : undefined}
                key={label}
              >
                <span>
                  {step > stepNumber ? (
                    <Check aria-hidden="true" size={15} weight="bold" />
                  ) : (
                    stepNumber
                  )}
                </span>
                {t(label)}
              </li>
            );
          })}
        </ol>

        <div className={styles.card}>
          {pageError && <InlineStatus tone="danger">{pageError}</InlineStatus>}
          {step === 1 && (
            <form noValidate onSubmit={continueFromName}>
              <h2>{t("newLibrary.nameTitle")}</h2>
              <p>{t("newLibrary.nameDescription")}</p>
              <FormField
                autoComplete="off"
                autoFocus
                error={nameError}
                label={t("newLibrary.nameLabel")}
                maxLength={128}
                name="libraryName"
                onChange={(event) => updateName(event.currentTarget.value)}
                required
                value={name}
              />
              <footer className={styles.actions}>
                <Button disabled>{t("newLibrary.back")}</Button>
                <Button type="submit" variant="primary">
                  {t("newLibrary.continue")}
                </Button>
              </footer>
            </form>
          )}

          {step === 2 && (
            <>
              <h2>{t("newLibrary.pathTitle")}</h2>
              <p>{t("newLibrary.pathDescription")}</p>
              <PathPicker
                breadcrumbs={breadcrumbs}
                currentPath={parent}
                fetchingMore={pathQuery.isFetchingNextPage}
                hasMore={pathQuery.hasNextPage}
                items={pathItems}
                locationName={location?.name ?? t("newLibrary.mediaRoot")}
                onEnter={setParent}
                onLoadMore={() => void loadNextPathPage()}
                onRetry={() => void refreshPaths()}
                onSelect={updateSelectedPath}
                selectedPath={selectedPath}
                state={
                  pathQuery.isPending
                    ? "loading"
                    : pathQuery.isError
                      ? "error"
                      : "ready"
                }
              />
              <footer className={styles.actions}>
                <Button onClick={() => setStep(1)}>{t("newLibrary.back")}</Button>
                <Button
                  disabled={selectedPath === null}
                  onClick={() => setStep(3)}
                  variant="primary"
                >
                  {t("newLibrary.continue")}
                </Button>
              </footer>
            </>
          )}

          {step === 3 && selectedPath !== null && (
            <>
              <h2>{t("newLibrary.reviewTitle")}</h2>
              <p>{t("newLibrary.reviewDescription")}</p>
              <dl className={styles.review}>
                <div>
                  <dt>{t("newLibrary.reviewName")}</dt>
                  <dd>{name}</dd>
                </div>
                <div>
                  <dt>{t("newLibrary.reviewPath")}</dt>
                  <dd>{displayPath(selectedPath)}</dd>
                </div>
                <div>
                  <dt>{t("newLibrary.reviewOriginals")}</dt>
                  <dd>{t("newLibrary.reviewOriginalsValue")}</dd>
                </div>
                <div>
                  <dt>{t("newLibrary.reviewScan")}</dt>
                  <dd>{t("newLibrary.reviewScanValue")}</dd>
                </div>
              </dl>
              <InlineStatus>
                <Info aria-hidden="true" size={18} />
                {t("newLibrary.relativePathNotice")}
              </InlineStatus>
              <footer className={styles.actions}>
                <Button onClick={() => setStep(2)}>{t("newLibrary.back")}</Button>
                <Button
                  loading={createMutation.isPending}
                  onClick={() => void submitLibrary()}
                  variant="primary"
                >
                  {t("newLibrary.createAndScan")}
                </Button>
              </footer>
            </>
          )}
        </div>
      </section>
    </AppShell>
  );
}

function PathPicker({
  breadcrumbs,
  currentPath,
  fetchingMore,
  hasMore,
  items,
  locationName,
  onEnter,
  onLoadMore,
  onRetry,
  onSelect,
  selectedPath,
  state,
}: {
  breadcrumbs: { name: string; relativePath: string }[];
  currentPath: string;
  fetchingMore: boolean;
  hasMore: boolean;
  items: LibraryPathEntry[];
  locationName: string;
  onEnter: (path: string) => void;
  onLoadMore: () => void;
  onRetry: () => void;
  onSelect: (path: string) => void;
  selectedPath: string | null;
  state: "error" | "loading" | "ready";
}) {
  const { t } = useLocale();

  return (
    <div className={styles.pathPicker}>
      <nav aria-label={t("newLibrary.pathBreadcrumbs")} className={styles.breadcrumbs}>
        <Database aria-hidden="true" size={18} />
        <button onClick={() => onEnter("")} type="button">
          /library
        </button>
        {breadcrumbs
          .filter((item) => item.relativePath)
          .map((item) => (
            <span key={item.relativePath}>
              <CaretRight aria-hidden="true" size={14} />
              <button onClick={() => onEnter(item.relativePath)} type="button">
                {item.name}
              </button>
            </span>
          ))}
        {breadcrumbs.length === 0 && locationName !== "/library" && (
          <span>{locationName}</span>
        )}
      </nav>
      <button
        className={`${styles.currentPath} ${
          selectedPath === currentPath ? styles.selectedPath : ""
        }`}
        onClick={() => onSelect(currentPath)}
        type="button"
      >
        <Database aria-hidden="true" size={22} />
        <span>
          <strong>{t("newLibrary.selectCurrent")}</strong>
          <small>{displayPath(currentPath)}</small>
        </span>
        {selectedPath === currentPath && (
          <Check aria-hidden="true" size={18} weight="bold" />
        )}
      </button>
      {state === "loading" && <LoadingState label={t("newLibrary.loadingPaths")} />}
      {state === "error" && (
        <ErrorState message={t("newLibrary.pathsFailed")} onRetry={onRetry} />
      )}
      {state === "ready" && items.length === 0 && (
        <p className={styles.emptyPaths}>{t("newLibrary.noDirectories")}</p>
      )}
      {state === "ready" &&
        items.map((item) => (
          <div
            className={`${styles.pathRow} ${
              selectedPath === item.relativePath ? styles.selectedPath : ""
            }`}
            key={item.relativePath}
          >
            <button
              className={styles.selectPath}
              disabled={!item.selectable}
              onClick={() => onSelect(item.relativePath)}
              type="button"
            >
              <FolderOpen aria-hidden="true" size={22} />
              <span>
                <strong>{item.name}</strong>
                <small>{pathDescription(item, t)}</small>
              </span>
              {selectedPath === item.relativePath && (
                <Check aria-hidden="true" size={18} weight="bold" />
              )}
            </button>
            {item.hasChildren && item.selectable && (
              <Button
                aria-label={t("newLibrary.openDirectory").replace("{name}", item.name)}
                onClick={() => onEnter(item.relativePath)}
                size="small"
                variant="quiet"
              >
                <CaretRight aria-hidden="true" size={18} />
              </Button>
            )}
          </div>
        ))}
      {state === "ready" && hasMore && (
        <Button loading={fetchingMore} onClick={onLoadMore}>
          {t("libraries.loadMore")}
        </Button>
      )}
    </div>
  );
}

function displayPath(relativePath: string) {
  return relativePath ? `/library/${relativePath}` : "/library";
}

function pathDescription(
  item: LibraryPathEntry,
  t: (key: MessageKey) => string,
) {
  if (item.selectable) {
    return item.hasChildren
      ? t("newLibrary.selectableWithChildren")
      : t("newLibrary.selectable");
  }
  if (!item.selectionBlockedReason) return t("newLibrary.blockedUnavailable");

  const message = t(blockedReasonMessage[item.selectionBlockedReason]);
  return item.conflictingLibraryName
    ? message.replace("{library}", item.conflictingLibraryName)
    : message.replace("{library}", t("libraries.title"));
}

function messageForCreateError(
  error: unknown,
  t: (key: MessageKey) => string,
) {
  if (!(error instanceof ApiError)) return t("newLibrary.createFailed");

  switch (error.code) {
    case "library_name_conflict":
      return t("newLibrary.nameConflict");
    case "library_path_overlap":
      return t("newLibrary.pathOverlap");
    case "library_root_unavailable":
      return t("newLibrary.rootUnavailable");
    case "library_root_symlink":
      return t("newLibrary.rootSymlink");
    case "library_root_mount_boundary":
      return t("newLibrary.rootMount");
    case "validation_failed":
    case "invalid_request":
      return t("newLibrary.validationFailed");
    default:
      return t("newLibrary.createFailed");
  }
}
