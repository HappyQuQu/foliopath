import type { ReactNode } from "react";
import {
  BrowserRouter,
  Navigate,
  Route,
  Routes,
  useNavigate,
  useParams,
} from "react-router-dom";

import { PublicLayout } from "../components/patterns/PublicLayout/PublicLayout";
import { LoadingState } from "../components/ui";
import {
  AuthPage,
  useAuthenticationStatusQuery,
  useLogoutMutation,
  useSessionQuery,
} from "../features/auth";
import { BrowsePage } from "../features/browse";
import {
  LibrariesPage,
  NewLibraryPage,
  ScanStatusPage,
} from "../features/libraries";
import { SystemUnavailablePage } from "../features/system/SystemUnavailablePage";
import { GeneralSettingsPage } from "../features/settings";
import { SearchPage } from "../features/search";
import { MediaViewerPage } from "../features/media";
import {
  messageForReadiness,
  useSystemReadinessQuery,
} from "../features/system/queries";
import { ApiError, isAuthenticationError } from "../lib/api/errors";
import { useLocale } from "../lib/i18n/LocaleProvider";
import { paths } from "./paths";

export function AppRouter() {
  return (
    <BrowserRouter>
      <AppRoutes />
    </BrowserRouter>
  );
}

export function AppRoutes() {
  return (
    <ReadinessGate>
      <Routes>
        <Route path={paths.root} element={<RootRoute />} />
        <Route path={paths.setup} element={<PublicAuthRoute mode="setup" />} />
        <Route path={paths.login} element={<PublicAuthRoute mode="login" />} />
        <Route path={paths.generalSettings} element={<ProtectedAccountRoute />} />
        <Route path={paths.browsePattern} element={<ProtectedBrowseRoute />} />
        <Route path={paths.mediaPattern} element={<ProtectedMediaRoute />} />
        <Route path={paths.search} element={<ProtectedSearchRoute />} />
        <Route
          path={paths.librarySearchPattern}
          element={<ProtectedSearchRoute />}
        />
        <Route path={paths.libraries} element={<ProtectedLibrariesRoute />} />
        <Route path={paths.newLibrary} element={<ProtectedNewLibraryRoute />} />
        <Route
          path={paths.libraryStatusPattern}
          element={<ProtectedScanStatusRoute />}
        />
        <Route path={paths.unavailable} element={<StandaloneUnavailableRoute />} />
        <Route path="*" element={<Navigate replace to={paths.root} />} />
      </Routes>
    </ReadinessGate>
  );
}

function ReadinessGate({ children }: { children: ReactNode }) {
  const { t } = useLocale();
  const readinessQuery = useSystemReadinessQuery();
  const { refetch: refreshReadiness } = readinessQuery;

  if (readinessQuery.isPending) return <RouteLoading />;
  if (readinessQuery.isError) {
    return (
      <PublicLayout>
        <SystemUnavailablePage
          message={t("error.serviceOffline")}
          onRetry={() => void refreshReadiness()}
        />
      </PublicLayout>
    );
  }
  if (readinessQuery.data.status === "not_ready") {
    return (
      <PublicLayout>
        <SystemUnavailablePage
          message={messageForReadiness(readinessQuery.data.reasonCode, t)}
          onRetry={() => void refreshReadiness()}
        />
      </PublicLayout>
    );
  }

  return children;
}

function RootRoute() {
  const statusQuery = useAuthenticationStatusQuery();
  const sessionQuery = useSessionQuery({
    enabled: statusQuery.isSuccess && !statusQuery.data.setupRequired,
  });

  if (statusQuery.isPending) return <RouteLoading />;
  if (statusQuery.isError) return <RouteError error={statusQuery.error} retry={statusQuery.refetch} />;
  if (statusQuery.data.setupRequired) return <Navigate replace to={paths.setup} />;
  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) return <Navigate replace to={paths.libraries} />;
  if (isAuthenticationError(sessionQuery.error)) return <Navigate replace to={paths.login} />;

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function PublicAuthRoute({ mode }: { mode: "login" | "setup" }) {
  const statusQuery = useAuthenticationStatusQuery();
  const sessionQuery = useSessionQuery({
    enabled: statusQuery.isSuccess && !statusQuery.data.setupRequired,
  });

  if (statusQuery.isPending) return <RouteLoading />;
  if (statusQuery.isError) return <RouteError error={statusQuery.error} retry={statusQuery.refetch} />;
  if (statusQuery.data.setupRequired && mode === "login") {
    return <Navigate replace to={paths.setup} />;
  }
  if (!statusQuery.data.setupRequired && mode === "setup") {
    return <Navigate replace to={paths.login} />;
  }
  if (mode === "login" && sessionQuery.isPending) return <RouteLoading />;
  if (mode === "login" && sessionQuery.isSuccess) {
    return <Navigate replace to={paths.libraries} />;
  }
  if (mode === "login" && sessionQuery.isError && !isAuthenticationError(sessionQuery.error)) {
    return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
  }

  return (
    <PublicLayout>
      <AuthPage mode={mode} />
    </PublicLayout>
  );
}

function ProtectedAccountRoute() {
  const navigate = useNavigate();
  const sessionQuery = useSessionQuery();
  const logoutMutation = useLogoutMutation();

  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) {
    return (
      <GeneralSettingsPage
        logoutPending={logoutMutation.isPending}
        onLogout={async () => {
          await logoutMutation.mutateAsync(sessionQuery.data.csrfToken);
          navigate(paths.login, { replace: true });
        }}
        session={sessionQuery.data}
      />
    );
  }
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function ProtectedLibrariesRoute() {
  const sessionQuery = useSessionQuery();

  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) return <LibrariesPage session={sessionQuery.data} />;
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function ProtectedBrowseRoute() {
  const { directoryId, libraryId } = useParams<{
    directoryId?: string;
    libraryId: string;
  }>();
  const sessionQuery = useSessionQuery();

  if (!libraryId) return <Navigate replace to={paths.libraries} />;
  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) {
    return (
      <BrowsePage
        libraryId={libraryId}
        session={sessionQuery.data}
        {...(directoryId ? { directoryId } : {})}
      />
    );
  }
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function ProtectedSearchRoute() {
  const { libraryId } = useParams<{ libraryId?: string }>();
  const sessionQuery = useSessionQuery();

  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) {
    return (
      <SearchPage
        session={sessionQuery.data}
        {...(libraryId ? { libraryId } : {})}
      />
    );
  }
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function ProtectedMediaRoute() {
  const { assetId, libraryId } = useParams<{
    assetId: string;
    libraryId: string;
  }>();
  const sessionQuery = useSessionQuery();

  if (!assetId || !libraryId) {
    return <Navigate replace to={paths.libraries} />;
  }
  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) {
    return <MediaViewerPage assetId={assetId} libraryId={libraryId} />;
  }
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function ProtectedNewLibraryRoute() {
  const sessionQuery = useSessionQuery();

  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) return <NewLibraryPage session={sessionQuery.data} />;
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function ProtectedScanStatusRoute() {
  const sessionQuery = useSessionQuery();

  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) return <ScanStatusPage session={sessionQuery.data} />;
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function RouteLoading() {
  const { t } = useLocale();

  return (
    <PublicLayout>
      <LoadingState label={t("error.confirmingSecurity")} />
    </PublicLayout>
  );
}

function RouteError({ error, retry }: { error: unknown; retry: () => unknown }) {
  const { t } = useLocale();
  const message =
    error instanceof ApiError
      ? t("error.serviceFailed")
      : t("error.pageFailed");

  return (
    <PublicLayout>
      <SystemUnavailablePage message={message} onRetry={() => void retry()} />
    </PublicLayout>
  );
}

function StandaloneUnavailableRoute() {
  return (
    <PublicLayout>
      <SystemUnavailablePage onRetry={() => window.location.assign(paths.root)} />
    </PublicLayout>
  );
}
