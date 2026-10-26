# Experimental only. No listener, app import, new deployment unit, or default activation.
FROM foliopath-inference:photo-tags-v2@sha256:52f4a40c9f0169e87783b138200e24c8e96492fccf244b9d329ec1e05e45f626
COPY spikes/ocr/*.py spikes/ocr/model-manifest.json /opt/ocr-spike/
ENV PYTHONPATH=/opt/ocr-spike:/opt/foliopath-inference MPLCONFIGDIR=/tmp/matplotlib
ENTRYPOINT ["python", "/opt/ocr-spike/probe.py"]
