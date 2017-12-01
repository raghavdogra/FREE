#ifndef CLASSIFICATION_H
#define CLASSIFICATION_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stddef.h>

typedef struct classifier_ctx classifier_ctx;

classifier_ctx* classifier_initialize(void);

const char* classifier_classify(
                                char* buffer, size_t length);

void classifier_destroy(classifier_ctx* ctx);

#ifdef __cplusplus
}
#endif

#endif // CLASSIFICATION_H
