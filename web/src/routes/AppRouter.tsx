import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";

import { PublicLayout } from "../components/patterns/PublicLayout/PublicLayout";
import { LoadingState } from "../components/ui";
import {
  AccountPage,
  AuthPage,
  useAuthenticationStatusQuery,
  useSessionQuery,
} from "../features/auth";
import { SystemUnavailablePage } from "../features/system/SystemUnavailablePage";
import { ApiError, isAuthenticationError } from "../lib/api/errors";
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
      <Routes>
        <Route path={paths.root} element={<RootRoute />} />
        <Route path={paths.setup} element={<PublicAuthRoute mode="setup" />} />
        <Route path={paths.login} element={<PublicAuthRoute mode="login" />} />
        <Route path={paths.generalSettings} element={<ProtectedAccountRoute />} />
        <Route path={paths.unavailable} element={<StandaloneUnavailableRoute />} />
        <Route path="*" element={<Navigate replace to={paths.root} />} />
      </Routes>
  );
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
  if (sessionQuery.isSuccess) return <Navigate replace to={paths.generalSettings} />;
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
    return <Navigate replace to={paths.generalSettings} />;
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
  const sessionQuery = useSessionQuery();

  if (sessionQuery.isPending) return <RouteLoading />;
  if (sessionQuery.isSuccess) return <AccountPage session={sessionQuery.data} />;
  if (isAuthenticationError(sessionQuery.error)) {
    return <Navigate replace to={`${paths.login}?reason=session_expired`} />;
  }

  return <RouteError error={sessionQuery.error} retry={sessionQuery.refetch} />;
}

function RouteLoading() {
  return (
    <PublicLayout>
      <LoadingState label="正在确认安全状态…" />
    </PublicLayout>
  );
}

function RouteError({ error, retry }: { error: unknown; retry: () => unknown }) {
  const message =
    error instanceof ApiError
      ? "FolioPath 暂时无法响应。原始媒体没有被修改。"
      : "页面暂时无法载入。";

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
