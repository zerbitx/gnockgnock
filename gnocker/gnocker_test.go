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
	client := http.Client{Timeout: time.Second * 10}
	port := 74656
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

			err := app.AddConfig(spec.Configurations{
				"configName": spec.Path{
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
			defer res.Body.Close()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(res.StatusCode).To(Equal(http.StatusTeapot))

			resBytes, err := ioutil.ReadAll(res.Body)
			Expect(err).ShouldNot(HaveOccurred())

			Expect(string(resBytes)).To(Equal(expectedResponse))
		})

		Context("a configuration is posted", func() {
			Context("JSON", func() {
				It("Responds as configured", func() {
					cfgFile, err := os.Open("../fixtures/post-config.json")
					Expect(err).ShouldNot(HaveOccurred())

					req, err := http.NewRequest(
						http.MethodPost,
						fmt.Sprintf("http://127.0.0.1:%d/config", port+1),
						cfgFile,
					)

					Expect(err).ShouldNot(HaveOccurred())
					res, err := client.Do(req)
					Expect(err).ShouldNot(HaveOccurred())
					defer res.Body.Close()

					Expect(res.StatusCode).To(Equal(http.StatusCreated))

					configTokens := map[string]string{}
					err = json.NewDecoder(res.Body).Decode(&configTokens)

					Expect(err).ShouldNot(HaveOccurred())

					req, err = http.NewRequest(
						http.MethodPost,
						fmt.Sprintf("http://127.0.0.1:%d/v1/gnock/gnock", port),
						nil)

					for _, tkn := range configTokens {
						req.Header.Add(TOKEN_HEADER, tkn)
					}

					res, err = client.Do(req)
					Expect(err).ShouldNot(HaveOccurred())
					Expect(res.StatusCode).To(Equal(http.StatusCreated))

					response, err := ioutil.ReadAll(res.Body)
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
					fmt.Sprintf("http://127.0.0.1:%d/config", port+1),
					cfgFile,
				)

				Expect(err).ShouldNot(HaveOccurred())
				res, err := client.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				defer res.Body.Close()

				Expect(res.StatusCode).To(Equal(http.StatusCreated))

				configTokens := map[string]string{}
				err = json.NewDecoder(res.Body).Decode(&configTokens)

				Expect(err).ShouldNot(HaveOccurred())

				req, err = http.NewRequest(
					http.MethodPost,
					fmt.Sprintf("http://127.0.0.1:%d/v1/gnock/gnock", port),
					nil)

				Expect(err).ShouldNot(HaveOccurred())

				for _, tkn := range configTokens {
					req.Header.Add(TOKEN_HEADER, tkn)
				}

				res, err = client.Do(req)
				Expect(err).ShouldNot(HaveOccurred())
				Expect(res.StatusCode).To(Equal(http.StatusCreated))

				response, err := ioutil.ReadAll(res.Body)
				Expect(string(response)).To(Equal("success"))
			})
		})
	})

	Context("Response body is templated", func() {
		It("Responds as configured", func() {
			pathWithParameters := "/ships/:class/:designation"
			expectedResponse := "The {{.designation}} is {{.class}} class."

			err := app.AddConfig(spec.Configurations{
				"templatedConfig": spec.Path{
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
				"withTTL": spec.Path{
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

			<-time.After(time.Second)

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
