package supportarchive

import (
	"io"

	"github.com/Dynatrace/dynatrace-operator/pkg/logd"
)

const (
	supportArchiveCollectorName = "supportarchiveoutput"
)

type supportArchiveOutputCollector struct {
	output io.Reader
	collectorCommon
}

func newSupportArchiveOutputCollector(log logd.Logger, supportArchive archiver, logBuffer io.Reader) collector {
	return supportArchiveOutputCollector{
		collectorCommon: collectorCommon{
			log:            log,
			supportArchive: supportArchive,
		},
		output: logBuffer,
	}
}

func (vc supportArchiveOutputCollector) Do() error {
	logInfof(vc.log, "Storing support archive output into %s", SupportArchiveOutputFileName)
	vc.supportArchive.addFile(SupportArchiveOutputFileName, vc.output)

	return nil
}
func (vc supportArchiveOutputCollector) Name() string {
	return supportArchiveCollectorName
}
