package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"

	tensorflow "github.com/tensorflow/tensorflow/tensorflow/go"
	"github.com/tensorflow/tensorflow/tensorflow/go/op"
)

const (
	recognizeResultCount = 3
	returnResultCount    = 2
)

var (
	graphFile  = "/model/tensorflow_inception_graph.pb"
	labelsFile = "/model/imagenet_comp_graph_label_strings.txt"
)

type Label struct {
	Label       string
	Probability float32
}

type Labels []Label

func (l Labels) Len() int {
	return len(l)
}
func (l Labels) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}
func (l Labels) Less(i, j int) bool {
	return l[i].Probability > l[j].Probability
}

var (
	modelGraph *tensorflow.Graph
	labels     []string
)

func main() {
	err := os.Setenv("TF_CPP_MIN_LOG_LEVEL", "2")
	if err != nil {
		log.Fatalln(err)
	}

	modelGraph, labels, err = loadModel()
	if err != nil {
		log.Fatalf("unable to load model: %v", err)
	}

	log.Println("Run RECOGNITION server ...")
	http.HandleFunc("/", mainHandler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func mainHandler(w http.ResponseWriter, r *http.Request) {
	normalizedImg, err := normalizeImage(r.Body)
	if err != nil {
		log.Fatalf("unable to make a normalizedImg from image: %v", err)
	}

	session, err := tensorflow.NewSession(modelGraph, nil)
	if err != nil {
		log.Fatalf("could not init session: %v", err)
	}

	outputRecognize, err := session.Run(
		map[tensorflow.Output]*tensorflow.Tensor{
			modelGraph.Operation("input").Output(0): normalizedImg,
		},
		[]tensorflow.Output{
			modelGraph.Operation("output").Output(0),
		},
		nil,
	)
	if err != nil {
		log.Fatalf("could not run inference: %v", err)
	}

	res := getTopLabels(labels, outputRecognize[0].Value().([][]float32)[0])
	log.Println("recognition result:")
	for _, l := range res {
		log.Printf("label: %s, probability: %.2f%%\n", l.Label, l.Probability*100)
	}

	msg := "Result set"
	for i := 0; i < returnResultCount; i++ {
		msg += fmt.Sprintf("%s (%.2f%%)\n", res[i].Label, res[i].Probability*100)
	}
	_, err = w.Write([]byte(msg))
	if err != nil {
		log.Fatalf("could not write server response: %v", err)
	}
}

func loadModel() (*tensorflow.Graph, []string, error) {
	model, err := ioutil.ReadFile(graphFile)
	if err != nil {
		return nil, nil, err
	}
	graph := tensorflow.NewGraph()
	if err := graph.Import(model, ""); err != nil {
		return nil, nil, err
	}

	labelsFile, err := os.Open(labelsFile)
	if err != nil {
		return nil, nil, err
	}
	defer labelsFile.Close()
	scanner := bufio.NewScanner(labelsFile)
	var labels []string
	for scanner.Scan() {
		labels = append(labels, scanner.Text())
	}

	return graph, labels, scanner.Err()
}

func getTopLabels(labels []string, probabilities []float32) []Label {
	var resultLabels []Label
	for i, p := range probabilities {
		if i >= len(labels) {
			break
		}
		resultLabels = append(resultLabels, Label{Label: labels[i], Probability: p})
	}
	sort.Sort(Labels(resultLabels))

	return resultLabels[:recognizeResultCount]
}

func normalizeImage(imgBody io.ReadCloser) (*tensorflow.Tensor, error) {
	var buf bytes.Buffer
	_, err := io.Copy(&buf, imgBody)
	if err != nil {
		return nil, err
	}

	tensor, err := tensorflow.NewTensor(buf.String())
	if err != nil {
		return nil, err
	}

	graph, input, output, err := getNormalizedGraph()
	if err != nil {
		return nil, err
	}

	session, err := tensorflow.NewSession(graph, nil)
	if err != nil {
		return nil, err
	}

	normalized, err := session.Run(
		map[tensorflow.Output]*tensorflow.Tensor{
			input: tensor,
		},
		[]tensorflow.Output{
			output,
		},
		nil)
	if err != nil {
		return nil, err
	}

	return normalized[0], nil
}

func getNormalizedGraph() (graph *tensorflow.Graph, input, output tensorflow.Output, err error) {
	s := op.NewScope()
	input = op.Placeholder(s, tensorflow.String)

	decode := op.DecodeJpeg(s, input, op.DecodeJpegChannels(3))

	output = op.Sub(s,

		op.ResizeBilinear(s,

			op.ExpandDims(s,
				op.Cast(s, decode, tensorflow.Float),
				op.Const(s.SubScope("make_batch"), int32(0))),
			op.Const(s.SubScope("size"), []int32{224, 224})),
		op.Const(s.SubScope("mean"), float32(117)))
	graph, err = s.Finalize()

	return graph, input, output, err
}
