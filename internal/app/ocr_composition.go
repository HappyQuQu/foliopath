package app

import (
	"github.com/HappyQuQu/foliopath/internal/inference/remoteml"
	"github.com/HappyQuQu/foliopath/internal/jobs"
	"github.com/HappyQuQu/foliopath/internal/media"
	"github.com/HappyQuQu/foliopath/internal/media/imagevips"
	"github.com/HappyQuQu/foliopath/internal/ocr"
	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
)

func composeOCR(database *databaseService, content *media.ContentService, client *remoteml.Client, config ocrDeploymentConfiguration, resources *resourcecontrol.Controller) (*ocr.Service, []aiBackgroundWorker, error) {
	bound, err := client.BindOCR(config.Identity)
	if err != nil {
		return nil, nil, err
	}
	analyzer, err := ocr.NewAnalyzer(content, imagevips.New(), ocrRuntime{client: bound})
	if err != nil {
		return nil, nil, err
	}
	fp, err := config.Identity.Fingerprint()
	if err != nil {
		return nil, nil, err
	}
	processor, err := ocr.NewWorker(database, analyzer, fp)
	if err != nil {
		return nil, nil, err
	}
	limited, err := resourcecontrol.LimitBackground(resources, processor)
	if err != nil {
		return nil, nil, err
	}
	signal := jobs.NewSignal()
	pool, err := jobs.NewWorkerPool(ocr.Queue{Repository: database}, limited, signal, jobs.WorkerOptions{Workers: 1})
	if err != nil {
		return nil, nil, err
	}
	service, err := ocr.NewService(database, config.Identity, signal)
	if err != nil {
		return nil, nil, err
	}
	return service, []aiBackgroundWorker{pool, service}, nil
}
