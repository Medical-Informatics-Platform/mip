package spawnapi

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	authenticationv1 "k8s.io/api/authentication/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

var _ = Describe("TokenReviewAuthenticator", func() {
	var (
		ns      string
		counter int
		auth    *TokenReviewAuthenticator
	)
	token := func(audience string) string {
		clientset, err := kubernetes.NewForConfig(cfg)
		Expect(err).NotTo(HaveOccurred())
		tr, err := clientset.CoreV1().ServiceAccounts(ns).CreateToken(ctx, "jupyterhub", &authenticationv1.TokenRequest{
			Spec: authenticationv1.TokenRequestSpec{Audiences: []string{audience}, ExpirationSeconds: ptr.To[int64](600)},
		}, metav1.CreateOptions{})
		Expect(err).NotTo(HaveOccurred())
		return tr.Status.Token
	}

	BeforeEach(func() {
		counter++
		ns = fmt.Sprintf("federation-auth-%d", counter)
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})).To(Succeed())
		Expect(k8sClient.Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: "jupyterhub", Namespace: ns}})).To(Succeed())
		auth = &TokenReviewAuthenticator{Client: k8sClient}
	})

	It("accepts a token for the mip-notebook-operator audience", func() {
		username, err := auth.Authenticate(ctx, token(Audience))
		Expect(err).NotTo(HaveOccurred())
		Expect(username).To(Equal("system:serviceaccount:" + ns + ":jupyterhub"))
	})
	It("rejects a token for another audience", func() {
		_, err := auth.Authenticate(ctx, token("https://kubernetes.default.svc"))
		Expect(err).To(MatchError(ErrUnauthenticated))
	})
	It("rejects a token that is not one", func() {
		_, err := auth.Authenticate(ctx, "not-a-token")
		Expect(err).To(MatchError(ErrUnauthenticated))
	})
	It("rejects an answer that does not confirm the audience", func() {
		auth.Client = interceptor.NewClient(k8sClient, interceptor.Funcs{
			Create: func(_ context.Context, _ client.WithWatch, obj client.Object, _ ...client.CreateOption) error {
				tr := obj.(*authenticationv1.TokenReview)
				tr.Status = authenticationv1.TokenReviewStatus{
					Authenticated: true,
					User:          authenticationv1.UserInfo{Username: "system:serviceaccount:" + ns + ":jupyterhub"},
					Audiences:     []string{"https://kubernetes.default.svc"},
				}
				return nil
			},
		})
		_, err := auth.Authenticate(ctx, "anything")
		Expect(err).To(MatchError(ErrUnauthenticated))
	})
})
