package authorizer

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/ghttp"
)

var _ = Describe("Token Retriever Tests", func() {
	var (
		server *ghttp.Server
	)

	BeforeEach(func() {
		server = ghttp.NewServer()
	})

	AfterEach(func() {
		//shut down the server between tests
		server.Close()
	})

	Context("Retrieve ARM Token", func() {
		It("Get ARM Token with Client ID Successfully", func() {
			armToken, err := getTestArmToken(time.Now().Add(time.Hour).Unix(), signingKey)
			Expect(err).ToNot(HaveOccurred())

			tokenResp := &tokenResponse{AccessToken: string(armToken)}

			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/", fmt.Sprintf("client_id=%s&resource=https://management.azure.com/&api-version=2018-02-01", testClientID)),
					ghttp.RespondWithJSONEncoded(200, tokenResp),
				))

			tr := newTestTokenRetriever(server.URL())
			token, err := tr.AcquireARMTokenMSI(context.Background(), testClientID)

			Expect(err).To(BeNil())
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(token).To(Equal(armToken))
		})

		It("Returns error when identity not found", func() {
			server.AppendHandlers(
				ghttp.CombineHandlers(
					ghttp.VerifyRequest("GET", "/", fmt.Sprintf("client_id=%s&resource=https://management.azure.com/&api-version=2018-02-01", testClientID)),
					ghttp.RespondWith(404, ""),
				))

			tr := newTestTokenRetriever(server.URL())
			token, err := tr.AcquireARMTokenMSI(context.Background(), testClientID)

			Expect(err).NotTo(BeNil())
			Expect(err.Error()).To(ContainSubstring("404"))
			Expect(server.ReceivedRequests()).Should(HaveLen(1))
			Expect(string(token)).To(Equal(""))
		})
	})
})

func newTestTokenRetriever(metadataEndpoint string) *TokenRetriever {
	return &TokenRetriever{
		metadataEndpoint:        metadataEndpoint,
		resourceManagerEndpoint: "https://management.azure.com/",
	}
}
