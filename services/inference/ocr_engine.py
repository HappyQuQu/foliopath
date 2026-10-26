"""Bounded OCR inference only. No paths to media, downloads or image transforms."""
import math
import cv2
import numpy as np
from rapidocr.ch_ppocr_det.utils import DBPostProcess
from pathlib import Path

from ocr_contract import MODEL, OUTPUT, InputLimit, InvalidOutput, normalize, valid_box


class BoundedPostProcess(DBPostProcess):
    def boxes_from_bitmap(self,pred,bitmap,dest_width,dest_height):
        count=len(cv2.findContours((bitmap*255).astype(np.uint8),cv2.RETR_LIST,cv2.CHAIN_APPROX_SIMPLE)[-2])
        if count>OUTPUT["max_contours"]: raise InputLimit()
        return super().boxes_from_bitmap(pred,bitmap,dest_width,dest_height)

def row_order(boxes):
    records=[]
    for index,box in enumerate(boxes):
        xs=[p[0] for p in box];ys=[p[1] for p in box]
        records.append((min(ys),min(xs),(min(ys)+max(ys))/2,max(ys)-min(ys),index))
    rows=[]
    for item in sorted(records):
        row=next((row for row in rows if abs(item[2]-row[0][2])<=min(item[3],row[0][3])/2),None)
        if row is None: rows.append([item])
        else: row.append(item)
    return [item[4] for row in rows for item in sorted(row,key=lambda v:(v[1],v[0],v[4]))]


def infer(cache,image):
    import immich_ml.models
    from immich_ml.models.ocr.detection import TextDetector
    from immich_ml.models.ocr.recognition import TextRecognizer
    from cache_snapshot import InvalidCache


    def no_download(*args,**kwargs): raise InvalidCache()
    TextDetector._download=no_download;TextRecognizer._download=no_download
    detector=TextDetector(MODEL,cache_dir=Path(cache)/"ocr"/MODEL,min_score=OUTPUT["min_detection"])
    detector.max_resolution=OUTPUT["detector_resolution"]
    detector.postprocess=BoundedPostProcess(thresh=.3,box_thresh=OUTPUT["min_detection"],max_candidates=OUTPUT["max_contours"],unclip_ratio=1.6,use_dilation=True,score_mode="fast")
    detector.sorted_boxes=lambda boxes:boxes
    detected=detector.predict(image)
    boxes,scores=detected["boxes"],detected["scores"]
    if len(boxes)>OUTPUT["max_lines"] or len(boxes)!=len(scores): raise InvalidOutput()
    if not len(boxes): return normalize([])
    crop_pixels=0
    for box in boxes.tolist():
        normalized=[value/(image.width if i%2==0 else image.height) for i,value in enumerate(sum(box,[]))]
        valid_box(normalized)
        width=max(math.dist(box[0],box[1]),math.dist(box[2],box[3]))
        height=max(math.dist(box[0],box[3]),math.dist(box[1],box[2]))
        crop_pixels+=math.ceil(width)*math.ceil(height)
    if crop_pixels>OUTPUT["max_crop_pixels"]: raise InputLimit()
    order=row_order(boxes.tolist())
    detected={"boxes":boxes[order],"scores":scores[order]}
    recognizer=TextRecognizer(MODEL,cache_dir=Path(cache)/"ocr"/MODEL,min_score=OUTPUT["min_recognition"])
    result=recognizer.predict(image,detected)
    text,quad,det,rec=result["text"],np.asarray(result["box"]).reshape(-1,8).tolist(),result["boxScore"].tolist(),result["textScore"].tolist()
    if len({len(text),len(quad),len(det),len(rec)})!=1: raise InvalidOutput()
    return normalize([{"text":t,"box":b,"detection_score":d,"recognition_score":r} for t,b,d,r in zip(text,quad,det,rec)])
