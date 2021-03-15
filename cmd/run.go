package cmd

import (
	"context"
	"github.com/dgraph-io/ristretto"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func init() {
	rootCmd.AddCommand(runCmd)
}

var runCmd = &cobra.Command{
	Use:   "run [flags] <program>",
	Short: "Run the beacon",
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := clientcmd.BuildConfigFromFlags("", viper.GetString("kubeconfig"))
		if err != nil {
			config, err = rest.InClusterConfig()
			if err != nil {
				return errors.Wrap(err, "kubeconfig not found and in-cluster failed to initialize")
			}
		}

		clientSet, err := kubernetes.NewForConfig(config)
		if err != nil {
			return err
		}

		accessTokenCache, err := ristretto.NewCache(&ristretto.Config{
			NumCounters: 1e7,
			MaxCost:     1 << 30,
			BufferItems: 64,
		})

		if err != nil {
			panic(err)
		}

		for {
			log.Debug().Msg("running update check")

			start := time.Now()
			imageCache := make(map[string]string)

			for _, namespace := range viper.GetStringSlice("namespaces") {
				list, err := clientSet.AppsV1().Deployments(namespace).List(context.TODO(), metav1.ListOptions{
					LabelSelector: viper.GetString("label") + "=true",
				})
				if err != nil {
					return err
				}

				log.Debug().Str("namespace", namespace).Int("deployments", len(list.Items)).Msg("checking for updates")

				for _, deployment := range list.Items {
					logDep := log.With().
						Str("namespace", deployment.Namespace).
						Str("deployment", deployment.Name).Logger()

					hasAlwaysPull := false
					requiresUpdate := false

					for _, container := range deployment.Spec.Template.Spec.Containers {
						if container.ImagePullPolicy == v1.PullAlways {
							hasAlwaysPull = true
						}

						label, ok := deployment.Spec.Template.Labels[viper.GetString("label")+"/"+container.Name]
						if !ok {
							logDep.Info().
								Str("container", container.Name).
								Msg("missing container image hash label")
							requiresUpdate = true
							break
						}

						digest, err := GetLatestImage(container.Image, &imageCache, accessTokenCache)
						if err != nil {
							return err
						}

						if label != digest {
							logDep.Info().
								Str("container", container.Name).
								Str("digest", digest).
								Str("old", label).
								Str("image", container.Image).
								Msg("updated image found")
							requiresUpdate = true
							break
						}
					}

					if !hasAlwaysPull {
						logDep.Error().
							Msg("deployment is labeled to be monitored, but no container is configured with Always Pull image policy")
					}

					if requiresUpdate {
						newLabels := make(map[string]string)
						for k, v := range deployment.Spec.Template.Labels {
							newLabels[k] = v
						}

						for _, container := range deployment.Spec.Template.Spec.Containers {
							digest, err := GetLatestImage(container.Image, &imageCache, accessTokenCache)
							if err != nil {
								return err
							}

							newLabels[viper.GetString("label")+"/"+container.Name] = digest
						}

						newDeployment := appsv1.Deployment{
							Spec: deployment.Spec,
						}

						newDeployment.Spec.Template.ObjectMeta.Labels = newLabels

						marshaledDeployment, err := json.Marshal(newDeployment)
						if err != nil {
							return err
						}

						_, err = clientSet.AppsV1().
							Deployments(deployment.Namespace).
							Patch(context.TODO(), deployment.Name, types.StrategicMergePatchType, marshaledDeployment, metav1.PatchOptions{})

						if err != nil {
							return err
						}

						logDep.Info().Msg("updated deployment")
					}
				}
			}

			log.Debug().Dur("took", time.Since(start)).Msg("update check completed")

			time.Sleep(viper.GetDuration("interval") - time.Since(start))
		}

		return nil
	},
}

func GetLatestImage(image string, cache *map[string]string, tokenCache *ristretto.Cache) (string, error) {
	if !strings.Contains(image, ":") {
		image += ":latest"
	}

	if !strings.Contains(image, "/") {
		image = "library/" + image
	}

	if strings.Count(image, "/") == 1 {
		image = "index.docker.io/" + image
	}

	if digest, ok := (*cache)[image]; ok {
		return digest, nil
	}

	parsedImage, err := url.Parse("https://" + image)
	if err != nil {
		return "", err
	}

	cleanName := strings.Trim(parsedImage.Path, "/")
	splitImage := strings.Split(cleanName, ":")

	requestURL := url.URL{
		Scheme: "https",
		Host:   parsedImage.Host,
		Path:   "/v2/" + splitImage[0] + "/manifests/" + splitImage[1],
	}

	request, err := http.NewRequest("HEAD", requestURL.String(), nil)
	if err != nil {
		return "", err
	}

	if token, ok := tokenCache.Get(requestURL.String()); ok {
		request.Header.Set("Authorization", "Bearer "+token.(string))
	}

	response, err := http.DefaultClient.Do(request)
	if response == nil {
		return "", err
	}
	_ = response.Body.Close()

	authHeader := response.Header.Get("Www-Authenticate")
	if authHeader != "" {
		basicData := strings.SplitN(authHeader, " ", 2)[1]
		data := make(map[string]string)
		for _, entry := range strings.Split(basicData, ",") {
			splitKV := strings.SplitN(entry, "=", 2)
			data[splitKV[0]] = strings.Trim(splitKV[1], "\"")
		}

		// TODO Cleanup
		authResponse, err := http.DefaultClient.Get(data["realm"] + "?service=" + data["service"] + "&scope=" + data["scope"])
		if err != nil {
			return "", err
		}
		defer authResponse.Body.Close()

		body, _ := ioutil.ReadAll(authResponse.Body)
		tokenResponse := struct {
			Token     string `json:"token"`
			ExpiresIn int    `json:"expires_in"`
		}{}

		if err = json.Unmarshal(body, &tokenResponse); err != nil {
			return "", err
		}

		request, err = http.NewRequest("HEAD", requestURL.String(), nil)
		if err != nil {
			return "", err
		}
		request.Header.Set("Authorization", "Bearer "+tokenResponse.Token)

		response, err = http.DefaultClient.Do(request)
		if response == nil {
			return "", err
		}
		_ = response.Body.Close()

		tokenCache.SetWithTTL(requestURL.String(), tokenResponse.Token, 1, time.Second*time.Duration(tokenResponse.ExpiresIn))
	}

	imageDigest := response.Header.Get("Docker-Content-Digest")
	if imageDigest == "" {
		return "", errors.New("image not found: " + image)
	}

	cleanDigest := strings.SplitN(imageDigest, ":", 2)[1]

	if len(cleanDigest) > 63 {
		cleanDigest = cleanDigest[:63]
	}

	(*cache)[image] = cleanDigest

	return cleanDigest, nil
}
