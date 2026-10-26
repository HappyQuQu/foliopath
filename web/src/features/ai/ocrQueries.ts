import { useEffect } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { cancelLibraryOCR, clearLibraryOCR, getLibraryOCR, requestLibraryOCR, updateLibraryOCR, type OCRJob } from "../../lib/api/ai";
import { searchKeys } from "../search";
import { aiKeys } from "./queries";

export function useLibraryOCR(libraryId:string){
 const client=useQueryClient();
 const query=useQuery({queryKey:aiKeys.ocr(libraryId),queryFn:()=>getLibraryOCR(libraryId),enabled:Boolean(libraryId),retry:false,
  refetchInterval:q=>q.state.data?.clearing || ["queued","running","cancelling"].includes(q.state.data?.latestJob?.state??"")?2000:10000});
 useEffect(()=>{if(query.data!==undefined)void client.invalidateQueries({queryKey:searchKeys.all});},[client,query.data]);
 return query;
}

export type OCRCommand = (
 {action:"settings";enabled:boolean} | {action:"run";mode:OCRJob["mode"];idempotencyKey:string} | {action:"cancel";jobId:string} | {action:"clear"});
type OCRAction = OCRCommand & {libraryId:string;csrfToken:string;etag:string};
export function useOCRAction(){
 const client=useQueryClient();
 return useMutation({mutationFn:async(input:OCRAction)=>{
  switch(input.action){
   case "settings":return updateLibraryOCR(input);
   case "run":return requestLibraryOCR(input);
   case "cancel":return cancelLibraryOCR(input);
   case "clear":return clearLibraryOCR(input);
  }
 },onSettled:async(_value,_error,input)=>{await Promise.all([client.invalidateQueries({queryKey:aiKeys.ocr(input.libraryId)}),client.invalidateQueries({queryKey:searchKeys.all})]);}});
}
