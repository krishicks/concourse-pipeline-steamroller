package steamroller_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	steamroller "github.com/krishicks/concourse-pipeline-steamroller"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	yaml "gopkg.in/yaml.v2"
)

var _ = Describe("Steamroller", func() {
	It("is a no-op when there's nothing to inline", func() {
		pipelineBytes := []byte(`jobs: []`)
		actualBytes, err := steamroller.Steamroll(nil, pipelineBytes)
		Expect(err).NotTo(HaveOccurred())

		var actual map[interface{}]interface{}
		err = yaml.Unmarshal(actualBytes, &actual)
		Expect(err).NotTo(HaveOccurred())

		expectedBytes := []byte(`jobs: []`)

		var expected map[interface{}]interface{}
		err = yaml.Unmarshal(expectedBytes, &expected)
		Expect(err).NotTo(HaveOccurred())

		Expect(actual).To(Equal(expected))
	})

	Context("when there is a file to inline", func() {
		var (
			resourceMap   map[string]string
			tmpdir        string
			pipelineBytes []byte
		)

		BeforeEach(func() {
			var err error
			tmpdir, err = ioutil.TempDir("", "steamroller")
			Expect(err).NotTo(HaveOccurred())

			taskFilePath := filepath.Join(tmpdir, "some-file")

			err = ioutil.WriteFile(taskFilePath, []byte("platform: linux"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			resourceMap = map[string]string{
				"some-place": tmpdir,
			}

			pipelineBytes = []byte(`---
jobs:
  - name: some-job
    plan:
      - task: some-task
        file: some-place/some-file
`)

		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		It("inlines the file", func() {
			actualBytes, err := steamroller.Steamroll(resourceMap, pipelineBytes)
			Expect(err).NotTo(HaveOccurred())

			var actual map[interface{}]interface{}
			err = yaml.Unmarshal(actualBytes, &actual)
			Expect(err).NotTo(HaveOccurred())

			expectedBytes := []byte(`---
jobs:
- name: some-job
  plan:
  - task: some-task
    config:
      platform: linux`)

			var expected map[interface{}]interface{}
			err = yaml.Unmarshal(expectedBytes, &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})
	})

	Context("when there is a task file to inline", func() {
		var (
			resourceMap   map[string]string
			tmpdir        string
			pipelineBytes []byte
		)

		BeforeEach(func() {
			var err error
			tmpdir, err = ioutil.TempDir("", "steamroller")
			Expect(err).NotTo(HaveOccurred())

			scriptFilePath := filepath.Join(tmpdir, "some-script.sh")
			err = ioutil.WriteFile(scriptFilePath, []byte(`#!/bin/bash
echo hi`), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			taskFilePath := filepath.Join(tmpdir, "some-task")
			err = ioutil.WriteFile(taskFilePath, []byte(`---
run:
  path: some-place/some-script.sh
`), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			resourceMap = map[string]string{
				"some-place": tmpdir,
			}

			pipelineBytes = []byte(`---
jobs:
  - name: some-job
    plan:
      - task: some-task
        file: some-place/some-task
`)

		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		It("inlines the file", func() {
			actualBytes, err := steamroller.Steamroll(resourceMap, pipelineBytes)
			Expect(err).NotTo(HaveOccurred())

			var actual map[interface{}]interface{}
			err = yaml.Unmarshal(actualBytes, &actual)
			Expect(err).NotTo(HaveOccurred())

			expectedBytes := []byte(`---
jobs:
- name: some-job
  plan:
  - task: some-task
    config:
      run:
        path: sh
        args:
        - -c
        - |
          cat > task.sh <<'EO_SH'
          #!/bin/bash
          echo hi
          EO_SH

          chmod +x task.sh
          ./task.sh
`)

			var expected map[interface{}]interface{}
			err = yaml.Unmarshal(expectedBytes, &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})
	})

	Context("when there is a file elsewhere in the job to inline", func() {
		var (
			resourceMap   map[string]string
			tmpdir        string
			pipelineBytes []byte
		)

		BeforeEach(func() {
			var err error
			tmpdir, err = ioutil.TempDir("", "steamroller")
			Expect(err).NotTo(HaveOccurred())

			taskFilePath := filepath.Join(tmpdir, "some-file")

			err = ioutil.WriteFile(taskFilePath, []byte("platform: linux"), os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			resourceMap = map[string]string{
				"some-place": tmpdir,
			}

			pipelineBytes = []byte(`---
jobs:
- name: some-job
  on_failure:
    task: some-task
    file: some-place/some-file
  plan:
  - task: some-other-task
`)

		})

		AfterEach(func() {
			os.RemoveAll(tmpdir)
		})

		It("inlines the file", func() {
			actualBytes, err := steamroller.Steamroll(resourceMap, pipelineBytes)
			Expect(err).NotTo(HaveOccurred())

			var actual map[interface{}]interface{}
			err = yaml.Unmarshal(actualBytes, &actual)
			Expect(err).NotTo(HaveOccurred())

			expectedBytes := []byte(`---
jobs:
- name: some-job
  on_failure:
    task: some-task
    config:
        platform: linux
  plan:
  - task: some-other-task
    `)

			var expected map[interface{}]interface{}
			err = yaml.Unmarshal(expectedBytes, &expected)
			Expect(err).NotTo(HaveOccurred())

			Expect(actual).To(Equal(expected))
		})
	})
})
