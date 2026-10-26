import {fireEvent,render,screen,waitFor,within} from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import {MemoryRouter} from "react-router-dom";
import {beforeEach,expect,it,vi} from "vitest";
import {LocaleProvider} from "../../../lib/i18n/LocaleProvider";
import {ApiError} from "../../../lib/api/errors";
import {PhotoFacesPanel} from "./PhotoFacesPanel";
const m=vi.hoisted(()=>({asset:vi.fn(),faces:vi.fn(),people:vi.fn(),person:vi.fn(),review:vi.fn(),refresh:vi.fn().mockResolvedValue({}),open:vi.fn()}));
const source={id:"person_source",name:"来源",revision:7},target={id:"person_target",name:"目标",revision:4};
const photo={id:"asset_photo",libraryId:"lib_1",name:"synthetic.png",kind:"image",sourceAvailability:"available",modifiedAt:"2026-10-25",sizeBytes:10};
const faces=[{faceId:"face_anonymous",assetId:"asset_photo",ordinal:1,revision:3,state:"anonymous_core",personId:null,region:{xPercent:10,yPercent:10,widthPercent:20,heightPercent:30}},{faceId:"face_assigned",assetId:"asset_photo",ordinal:2,revision:5,state:"assigned",personId:source.id,region:{xPercent:60,yPercent:10,widthPercent:20,heightPercent:30}}];
const snapshot=(data:unknown)=>({data,isPending:false,isFetching:false,isError:false,refetch:m.refresh});
vi.mock("../../media",()=>({useAssetQuery:m.asset}));
vi.mock("../queries",()=>({useAssetFacesQuery:m.faces,usePeopleQuery:m.people,usePersonQuery:m.person,useApplyFaceReviewMutation:()=>({mutateAsync:m.review})}));
function panel(){return <LocaleProvider><MemoryRouter><PhotoFacesPanel assetId="asset_photo" onAsset={m.open} session={{csrfToken:"synthetic",expiresAt:"2099-01-01",administrator:{id:"admin",username:"fixture",displayName:"Synthetic"}}}/></MemoryRouter></LocaleProvider>;}
function show(){const value=render(panel());fireEvent.load(screen.getByRole("img",{name:photo.name}));return value;}
beforeEach(()=>{vi.clearAllMocks();localStorage.setItem("foliopath.preferences.v1",'{"locale":"zh-CN"}');m.asset.mockReturnValue(snapshot(photo));m.faces.mockReturnValue(snapshot({items:faces}));m.people.mockReturnValue(snapshot({items:[source,target],nextCursor:null}));m.person.mockImplementation(id=>snapshot(id===source.id?source:id===target.id?target:undefined));m.review.mockResolvedValue({revision:2});});
it("requires an explicit face and cancels without writing",async()=>{
 show();const u=userEvent.setup();expect(screen.getAllByRole("radio").every(r=>!(r as HTMLInputElement).checked)).toBe(true);expect(screen.getByRole("button",{name:"排除此人脸"})).toBeDisabled();
 await u.click(screen.getByRole("radio",{name:"人脸 2 · 已归类"}));await u.click(screen.getByRole("button",{name:"排除此人脸"}));const d=screen.getByRole("dialog");expect(within(d).getByText(/来源人物： 来源/)).toBeVisible();await u.keyboard("{Escape}");expect(m.review).not.toHaveBeenCalled();
});
it("freezes face, source and target revisions before a confirmed move",async()=>{
 const view=show();const u=userEvent.setup();await u.click(screen.getByRole("radio",{name:"人脸 2 · 已归类"}));await u.selectOptions(screen.getByLabelText("目标人物"),target.id);await u.click(screen.getByRole("button",{name:"核对归入人物"}));
 m.person.mockImplementation(id=>snapshot(id===source.id?{...source,revision:99}:id===target.id?{...target,revision:99}:undefined));view.rerender(panel());await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));
 expect(m.review).toHaveBeenCalledWith(expect.objectContaining({review:{action:"assign_face",faceId:"face_assigned",expectedFaceRevision:5,personId:target.id,expectedPersonRevision:4,sourcePersonId:source.id,expectedSourceRevision:7}}));await waitFor(()=>expect(screen.getByText(/所选人脸已保存/)).toBeVisible());
});
it("fences assigned exclusion and leaves anonymous exclusion unbound",async()=>{
 show();const u=userEvent.setup();await u.click(screen.getByRole("radio",{name:"人脸 2 · 已归类"}));await u.click(screen.getByRole("button",{name:"排除此人脸"}));await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));
 expect(m.review.mock.calls[0]?.[0].review).toEqual({action:"exclude_face",faceId:"face_assigned",expectedFaceRevision:5,sourcePersonId:source.id,expectedSourceRevision:7});await waitFor(()=>expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
 await u.click(screen.getByRole("radio",{name:"人脸 1 · 待确认"}));await u.click(screen.getByRole("button",{name:"排除此人脸"}));await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));expect(m.review.mock.calls[1]?.[0].review).toEqual({action:"exclude_face",faceId:"face_anonymous",expectedFaceRevision:3});
});
it("retains the exact operation key after an uncertain response",async()=>{
 m.review.mockRejectedValueOnce(new Error("response lost"));show();const u=userEvent.setup();await u.click(screen.getByRole("radio",{name:"人脸 2 · 已归类"}));await u.click(screen.getByRole("button",{name:"从人物中移除"}));await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));await waitFor(()=>expect(within(screen.getByRole("dialog")).getByText(/保存结果尚未确认/)).toBeVisible());
 await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));expect(m.review.mock.calls[1]?.[0]).toEqual(m.review.mock.calls[0]?.[0]);expect(m.review.mock.calls[0]?.[0].review).toEqual({action:"split_face",faceId:"face_assigned",expectedFaceRevision:5,sourcePersonId:source.id,expectedSourceRevision:7});
});
it("does not resend after a curation conflict",async()=>{
 m.review.mockRejectedValue(new ApiError({code:"face_constraint_conflict",message:"changed",status:409,requestId:undefined}));show();const u=userEvent.setup();await u.click(screen.getByRole("radio",{name:"人脸 1 · 待确认"}));await u.click(screen.getByRole("button",{name:"排除此人脸"}));await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));await waitFor(()=>expect(screen.getByText(/当前整理状态已变化/)).toBeVisible());expect(m.review).toHaveBeenCalledTimes(1);expect(screen.queryByRole("dialog")).not.toBeInTheDocument();expect(screen.getAllByRole("radio").every(r=>!(r as HTMLInputElement).checked)).toBe(true);
});
it("blocks editing when the photo or owner state is unavailable",async()=>{
 show();fireEvent.error(screen.getByRole("img",{name:photo.name}));expect(screen.getByText(/照片未能载入/)).toBeVisible();expect(screen.getAllByRole("radio").every(r=>(r as HTMLInputElement).disabled)).toBe(true);expect(m.review).not.toHaveBeenCalled();
});
function separateFaces(peers: string[][] | null = [[], []]) {
 m.faces.mockReturnValue(snapshot({items:faces.map((face,index)=>({...face,state:"assigned",personId:index===0?source.id:target.id,...(peers?{cannotLinkFaceIds:peers[index]}:{})}))}));
}
async function choosePair(){const u=userEvent.setup();await u.click(screen.getByRole("radio",{name:"人脸 1 · 已归类"}));await u.click(screen.getByText("防止误合并",{selector:"summary"}));await u.selectOptions(screen.getByLabelText("另一张已归类的人脸"),"face_assigned");return u;}
it("separates only the explicit pair and freezes both people and detector revisions",async()=>{
 separateFaces();const view=show();const u=await choosePair();const trigger=screen.getByRole("button",{name:"标记为不同人物"});await u.click(trigger);expect(m.review).not.toHaveBeenCalled();await u.keyboard("{Escape}");expect(trigger).toHaveFocus();expect(m.review).not.toHaveBeenCalled();await u.click(trigger);
 m.person.mockImplementation(id=>snapshot(id===source.id?{...source,revision:99}:id===target.id?{...target,revision:99}:undefined));view.rerender(panel());await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));
 expect(m.review.mock.calls[0]?.[0].review).toEqual({action:"cannot_link",leftFaceId:"face_anonymous",rightFaceId:"face_assigned",expectedLeftRevision:3,expectedRightRevision:5,sourcePersonId:source.id,targetPersonId:target.id,expectedSourceRevision:7,expectedTargetRevision:4});await waitFor(()=>expect(screen.queryByRole("dialog")).not.toBeInTheDocument());
});
it("keeps the same pair and key through an uncertain save",async()=>{
 separateFaces();m.review.mockRejectedValueOnce(new Error("lost response"));show();const u=await choosePair();await u.click(screen.getByRole("button",{name:"标记为不同人物"}));await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));await waitFor(()=>expect(within(screen.getByRole("dialog")).getByText(/保存结果尚未确认/)).toBeVisible());await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));expect(m.review.mock.calls[1]?.[0]).toEqual(m.review.mock.calls[0]?.[0]);
});
it.each([null,[["face_assigned"],["face_anonymous"]],[["face_assigned"],[]]])("blocks unknown, existing or inconsistent pair state %j",async peers=>{
 separateFaces(peers);show();await choosePair();expect(screen.getByRole("button",{name:"标记为不同人物"})).toBeDisabled();expect(m.review).not.toHaveBeenCalled();
});
it("blocks separate-person writes after curation conflict without resending",async()=>{
 separateFaces();m.review.mockRejectedValue(new ApiError({code:"face_constraint_conflict",message:"changed",status:409,requestId:undefined}));show();const u=await choosePair();await u.click(screen.getByRole("button",{name:"标记为不同人物"}));await u.click(within(screen.getByRole("dialog")).getByRole("button",{name:"确认保存"}));await waitFor(()=>expect(screen.queryByRole("dialog")).not.toBeInTheDocument());expect(m.review).toHaveBeenCalledTimes(1);expect(screen.getAllByRole("radio").every(r=>!(r as HTMLInputElement).checked)).toBe(true);
});
it("does not treat two faces assigned to the same person as a separate pair",async()=>{
 m.faces.mockReturnValue(snapshot({items:faces.map(face=>({...face,state:"assigned",personId:source.id,cannotLinkFaceIds:[]}))}));show();const u=userEvent.setup();await u.click(screen.getByRole("radio",{name:"人脸 1 · 已归类"}));await u.click(screen.getByText("防止误合并",{selector:"summary"}));expect(screen.queryByLabelText("另一张已归类的人脸")).not.toBeInTheDocument();expect(m.review).not.toHaveBeenCalled();
});
