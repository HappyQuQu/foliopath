import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterAll, afterEach, beforeEach, expect, it, vi } from "vitest";
import { MemoryRouter } from "react-router-dom";
import type { OCRStatus } from "../../../lib/api/ai";
import { LocaleProvider } from "../../../lib/i18n/LocaleProvider";
import { aiKeys } from "../queries";
import { OCRSettings } from "./OCRSettings";

// Node's Request needs an absolute URL; browsers resolve these API paths.
const nativeRequest=vi.hoisted(()=>{
 const Native=globalThis.Request;
 globalThis.Request=class extends Native {constructor(input:RequestInfo|URL,init?:RequestInit){super(typeof input==="string"?new URL(input,"http://localhost"):input,init);}};
 return Native;
});
afterAll(()=>{globalThis.Request=nativeRequest;});
const original:OCRStatus={libraryId:"lib_2",enabled:true,clearing:false,revision:3,epoch:1,fingerprint:"a".repeat(64),expectedFingerprint:"a".repeat(64),state:"degraded",coverage:{eligible:5,ready:2,empty:1,failed:1,pending:2,stale:1},latestJob:{id:"ocrjob_fixture_123",libraryId:"lib_2",epoch:1,fingerprint:"a".repeat(64),mode:"missing",state:"failed",revision:2,recorded:3,failed:1,attempts:1,errorCode:"model_unavailable"}};
let value:OCRStatus;let requests:Request[];let status:number;let client:QueryClient;
const fetchFixture=vi.fn(async(input:RequestInfo|URL)=>{
 const r=input as Request;requests.push(r);const path=new URL(r.url).pathname;
 if(status!==200)return Response.json({error:{code:status===404?"ocr_not_found":"internal_error",message:"Synthetic error"}},{status});
 if(r.method==="GET"&&path.endsWith("/ocr"))return Response.json(value,{headers:{ETag:'"ocr-library-2-r3"'}});
 if(r.method==="GET"&&path.endsWith("/ocr/jobs/ocrjob_fixture_123"))return Response.json(value.latestJob,{headers:{ETag:'"ocr-job-ocrjob_fixture_123-r2"'}});
 if(r.method==="PUT")return Response.json(value,{headers:{ETag:'"ocr-library-2-r4"'}});
 if(path.endsWith("/clear"))return Response.json({...value,clearing:true,enabled:false,state:"clearing"},{status:202,headers:{ETag:'"ocr-library-2-r4"'}});
 return Response.json({...value.latestJob,state:"queued"},{status:202,headers:{ETag:'"ocr-job-ocrjob_fixture_123-r2"'}});
});
beforeEach(()=>{
 value=structuredClone(original);requests=[];status=200;fetchFixture.mockClear();vi.stubGlobal("fetch",fetchFixture);
 window.localStorage.setItem("foliopath.preferences.v1",'{"locale":"zh-CN"}');
 client=new QueryClient({defaultOptions:{queries:{retry:false},mutations:{retry:false}}});
});
afterEach(()=>{cleanup();client.clear();vi.unstubAllGlobals();});
function show(mode:"settings"|"tasks"="tasks",id="lib_2"){
 return render(<QueryClientProvider client={client}><MemoryRouter><LocaleProvider><OCRSettings libraryId={id} csrfToken="csrf-test" offline={false} mode={mode} onSettings={vi.fn()} onTasks={vi.fn()}/></LocaleProvider></MemoryRouter></QueryClientProvider>);
}
it("renders actual coverage, blank successes and task error without a generic fake operation",async()=>{
 show();expect(await screen.findByText("2/5 张已识别 · 1 张失败")).toBeVisible();
 expect(screen.getByText(/文字识别暂不可用/)).toBeVisible();expect(screen.getByText(/未识别到文字 1 张/)).toBeVisible();
 expect(screen.getByText(/已处理 3 张 · 1 张失败/)).toBeVisible();expect(screen.queryByRole("switch")).not.toBeInTheDocument();
 expect(requests.every(r=>r.method==="GET"&&r.url.includes("/lib_2/ai/ocr"))).toBe(true);
});
it("uses exact library ETag and CSRF for the switch without starting a job",async()=>{
 show("settings");await userEvent.click(await screen.findByRole("switch",{name:"为此媒体库识别文字"}));
 await waitFor(()=>expect(requests.some(r=>r.method==="PUT")).toBe(true));
 const r=requests.find(r=>r.method==="PUT")!;expect(r.headers.get("If-Match")).toBe('"ocr-library-2-r3"');expect(r.headers.get("X-CSRF-Token")).toBe("csrf-test");expect(await r.clone().json()).toEqual({enabled:false});expect(requests.some(r=>r.method==="POST")).toBe(false);
});
it("fills missing with generated client and only retries sources after explicit confirmation",async()=>{
 const user=userEvent.setup();show();await user.click(await screen.findByRole("button",{name:"补齐文字识别"}));
 await waitFor(()=>expect(requests.some(r=>r.method==="POST")).toBe(true));
 const missing=requests.find(r=>r.method==="POST")!;expect(await missing.clone().json()).toEqual({mode:"missing"});expect(missing.headers.get("Idempotency-Key")).toBeTruthy();
 await user.click(screen.getByRole("button",{name:"更多：文字识别"}));await user.click(screen.getByRole("button",{name:"重试失败图片"}));
 const dialog=screen.getByRole("dialog");expect(dialog).toHaveTextContent("损坏或空文件需要先修复");expect(requests.filter(r=>r.method==="POST")).toHaveLength(1);
 await user.click(within(dialog).getByRole("button",{name:"重试失败图片"}));await waitFor(()=>expect(requests.filter(r=>r.method==="POST")).toHaveLength(2));
 expect(await requests.filter(r=>r.method==="POST")[1]!.clone().json()).toEqual({mode:"retry_failed",confirmation:"retry_failed_ocr"});
});
it("clear confirms OCR-only scope and uses no tag/face/semantic endpoints",async()=>{
 const user=userEvent.setup();show();await user.click(await screen.findByRole("button",{name:"更多：文字识别"}));await user.click(screen.getByRole("button",{name:"清理识别文字"}));
 const dialog=screen.getByRole("dialog");expect(dialog).toHaveTextContent("原图片、标签、人物和其他搜索结果保留");expect(requests.every(r=>r.method==="GET")).toBe(true);
 await user.click(within(dialog).getByRole("button",{name:"清理识别文字"}));await waitFor(()=>expect(requests.some(r=>r.method==="POST")).toBe(true));
 const r=requests.find(r=>r.method==="POST")!;expect(r.url).toMatch(/\/lib_2\/ai\/ocr\/clear$/);expect(await r.clone().json()).toEqual({confirmation:"clear_ocr_results"});
});
it("reads the actual job ETag before cancelling and retains completed results",async()=>{
 value.state="running";value.latestJob!.state="running";value.latestJob!.errorCode=null;show();await userEvent.click(await screen.findByRole("button",{name:"停止文字识别"}));
 await waitFor(()=>expect(requests.some(r=>r.method==="POST")).toBe(true));const r=requests.find(r=>r.method==="POST")!;expect(r.url).toMatch(/ocrjob_fixture_123\/cancel$/);expect(r.headers.get("If-Match")).toBe('"ocr-job-ocrjob_fixture_123-r2"');expect(await r.clone().json()).toEqual({});
});
it.each([404,503])("does not expose stale completion or write actions for HTTP %s",async code=>{
 status=code;show();await screen.findByText(code===404?/文字识别暂不可用/:"暂时无法读取此项任务的处理状态。");
 expect(screen.getByRole("button",{name:"补齐文字识别"})).toBeDisabled();expect(screen.queryByRole("progressbar")).not.toBeInTheDocument();expect(requests.some(r=>r.method!=="GET")).toBe(false);
});

it("hides loaded success after polling fails and does not replay a conflicting write",async()=>{
 show("settings");expect(await screen.findByText("2/5 张已识别 · 1 张失败")).toBeVisible();
 fetchFixture.mockImplementationOnce(async(input:RequestInfo|URL)=>{requests.push(input as Request);return Response.json({error:{code:"precondition_failed",message:"Changed"}},{status:412});});
 await userEvent.click(screen.getByRole("switch"));expect(await screen.findByText("状态已变化，已刷新。请核对当前状态后重试。")).toBeVisible();
 expect(requests.filter(r=>r.method==="PUT")).toHaveLength(1);
 status=503;await act(()=>client.invalidateQueries({queryKey:aiKeys.ocr("lib_2")}));
 expect(await screen.findByText("暂时无法读取此项任务的处理状态。")).toBeVisible();expect(screen.queryByText("2/5 张已识别 · 1 张失败")).not.toBeInTheDocument();expect(screen.queryByRole("switch")).not.toBeInTheDocument();
});
