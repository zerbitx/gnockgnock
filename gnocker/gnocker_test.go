package gnocker

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/zerbitx/gnockgnock/spec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Gnocker", func() {
	client := http.Client{Timeout: time.Millisecond * 100}
	port := 1701
	var app *gnocker

	BeforeSuite(func() {
		logrus.SetOutput(ioutil.Discard)
		app = New(WithPort(port))

		go func() {
			err := app.Start()
			Expect(err).ShouldNot(HaveOccurred())
		}()

		// Wait for the server to start
		Eventually(func() error {
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://127.0.0.1:%d", port),
				nil)

			Expect(err).ShouldNot(HaveOccurred())

			_, err = client.Do(req)
			return err
		}).ShouldNot(HaveOccurred())
	})

	AfterSuite(func() {
		client.CloseIdleConnections()
		Expect(app.Shutdown()).ShouldNot(HaveOccurred())
	})

	Context("Nothing is configured", func() {
		It("Responds with a 404", func() {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://127.0.0.1:%d/anything", port), nil)

			Expect(err).ShouldNot(HaveOccurred())

			res, err := client.Do(req)

			Expect(err).ShouldNot(HaveOccurred())
			defer res.Body.Close()

			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
		})
	})

	Context("With a route configuration", func() {
		It("Responds as configured", func() {
			path := "/any/old/path"
			expectedResponse := "success"
			expectedHeader := "X-GNOCK-TEST"
			expectedHeaderValue := "A+"

			err := app.AddConfig(spec.Configurations{
				"configName": spec.Configuration{
					Paths: map[string]spec.Methods{
						path: map[string]spec.Method{
							http.MethodGet: {
								ResponseHeaders: []map[string]string{
									{
										expectedHeader: expectedHeaderValue,
									},
								},
								ResponseBody: expectedResponse,
								StatusCode:   http.StatusTeapot,
							},
						},
					},
				},
			})

			Expect(err).ShouldNot(HaveOccurred())
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://127.0.0.1:%d%s", port, path),
				nil,
			)

			Expect(err).ShouldNot(HaveOccurred())
			res, err := client.Do(req)
			defer res.Body.Close()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusTeapot))

			resBytes, err := ioutil.ReadAll(res.Body)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(string(resBytes)).To(Equal(expectedResponse))
			Expect(res.Header.Get(expectedHeader)).To(Equal(expectedHeaderValue))
		})

		Context("a configuration is posted", func() {
			Context("JSON", func() {
				It("Responds as configured", func() {
					cfgFile, err := os.Open("../fixtures/post-config.json")
					Expect(err).ShouldNot(HaveOccurred())

					req, err := http.NewRequest(
						http.MethodPost,
						fmt.Sprintf("http://127.0.0.1:%d", port+1),
						cfgFile,
					)

					Expect(err).ShouldNot(HaveOccurred())

					res, err := client.Do(req)

					Expect(err).ShouldNot(HaveOccurred())
					defer res.Body.Close()

					Expect(res.StatusCode).To(Equal(http.StatusCreated))

					var configs []string
					err = json.NewDecoder(res.Body).Decode(&configs)

					Expect(err).ShouldNot(HaveOccurred())

					req, err = http.NewRequest(
						http.MethodPost,
						fmt.Sprintf("http://127.0.0.1:%d/v1/gnock/gnock", port),
						nil)

					Expect(err).ShouldNot(HaveOccurred())

					for _, c := range configs {
						req.Header.Add(GnockerHeader, c)
					}

					res, err = client.Do(req)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(res.StatusCode).To(Equal(http.StatusCreated))

					response, err := ioutil.ReadAll(res.Body)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(string(response)).To(Equal("success"))
				})
			})
		})

		Context("YAML", func() {
			It("Responds as configured", func() {
				cfgFile, err := os.Open("../fixtures/post-config.yaml")
				Expect(err).ShouldNot(HaveOccurred())

				req, err := http.NewRequest(
					http.MethodPost,
					fmt.Sprintf("http://127.0.0.1:%d", port+1),
					cfgFile,
				)

				Expect(err).ShouldNot(HaveOccurred())
				res, err := client.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				defer res.Body.Close()

				Expect(res.StatusCode).To(Equal(http.StatusCreated))

				var configs []string
				err = json.NewDecoder(res.Body).Decode(&configs)

				Expect(err).ShouldNot(HaveOccurred())

				req, err = http.NewRequest(
					http.MethodPost,
					fmt.Sprintf("http://127.0.0.1:%d/v1/gnock/gnock", port),
					nil)

				Expect(err).ShouldNot(HaveOccurred())

				for _, c := range configs {
					req.Header.Add(GnockerHeader, c)
				}

				res, err = client.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(res.StatusCode).To(Equal(http.StatusCreated))

				response, err := ioutil.ReadAll(res.Body)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(string(response)).To(Equal("success"))
			})
		})
	})

	Context("Response body is templated", func() {
		It("Responds as configured", func() {
			pathWithParameters := "/ships/:class/:designation"
			expectedResponse := "The {{.designation}} is {{.class}} class."

			err := app.AddConfig(spec.Configurations{
				"templatedConfig": spec.Configuration{
					Paths: map[string]spec.Methods{
						pathWithParameters: map[string]spec.Method{
							http.MethodGet: {
								ResponseBodyTemplate: expectedResponse,
								StatusCode:           http.StatusOK,
							},
						},
					},
				},
			})

			Expect(err).ShouldNot(HaveOccurred())
			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://127.0.0.1:%d%s", port, "/ships/Galaxy/Enterprise"),
				nil,
			)

			Expect(err).ShouldNot(HaveOccurred())
			res, err := client.Do(req)
			defer res.Body.Close()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusOK))

			resBytes, err := ioutil.ReadAll(res.Body)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(string(resBytes)).To(Equal("The Enterprise is Galaxy class."))
		})
	})

	Context("With a TTL", func() {
		It("Should respond according to the config until the TTL, then 404", func() {
			path := "/with/ttl"
			expectedResponse := "success"

			err := app.AddConfig(spec.Configurations{
				"withTTL": spec.Configuration{
					TTL: "1s",
					Paths: map[string]spec.Methods{
						path: map[string]spec.Method{
							http.MethodGet: {
								ResponseBody: expectedResponse,
								StatusCode:   http.StatusTeapot,
							},
						},
					},
				},
			})

			Expect(err).ShouldNot(HaveOccurred())

			req, err := http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://127.0.0.1:%d%s", port, path),
				nil,
			)

			Expect(err).ShouldNot(HaveOccurred())
			res, err := client.Do(req)
			Expect(err).ShouldNot(HaveOccurred())
			defer res.Body.Close()

			Expect(res.StatusCode).To(Equal(http.StatusTeapot))

			resBytes, err := ioutil.ReadAll(res.Body)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(string(resBytes)).To(Equal(expectedResponse))

			time.Sleep(time.Second)

			req, err = http.NewRequest(
				http.MethodGet,
				fmt.Sprintf("http://127.0.0.1:%d%s", port, path),
				nil,
			)

			Expect(err).ShouldNot(HaveOccurred())

			res, err = client.Do(req)
			Expect(err).ShouldNot(HaveOccurred())
			defer res.Body.Close()

			Expect(res.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
